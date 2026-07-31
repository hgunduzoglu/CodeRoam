package dev.coderoam.coderoam

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.nio.ByteBuffer
import java.nio.charset.CodingErrorAction
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey

internal class MobileIdentityStorage(context: Context) {
    private val preferences =
        context.getSharedPreferences(preferencesName, Context.MODE_PRIVATE)

    fun read(): String? = synchronized(storageLock) {
        val encoded = preferences.getString(recordKey, null) ?: return@synchronized null
        decrypt(encoded)
    }

    fun createIfAbsent(value: String): Boolean {
        require(value.length <= maximumRecordBytes)
        val plaintext = value.toByteArray(Charsets.UTF_8)
        try {
            require(plaintext.size <= maximumRecordBytes)
            return synchronized(storageLock) {
                if (preferences.contains(recordKey)) {
                    return@synchronized false
                }
                val encrypted = encrypt(plaintext)
                if (!preferences.edit().putString(recordKey, encrypted).commit()) {
                    throw IllegalStateException("identity storage commit failed")
                }
                true
            }
        } finally {
            plaintext.fill(0)
        }
    }

    private fun encrypt(plaintext: ByteArray): String {
        val cipher = Cipher.getInstance(cipherTransformation)
        cipher.init(Cipher.ENCRYPT_MODE, getOrCreateKey())
        val initializationVector = cipher.iv
        val ciphertext = cipher.doFinal(plaintext)
        val record = ByteArray(2 + initializationVector.size + ciphertext.size)
        try {
            record[0] = recordVersion
            record[1] = initializationVector.size.toByte()
            initializationVector.copyInto(record, destinationOffset = 2)
            ciphertext.copyInto(record, destinationOffset = 2 + initializationVector.size)
            return Base64.encodeToString(record, Base64.NO_WRAP)
        } finally {
            initializationVector.fill(0)
            ciphertext.fill(0)
            record.fill(0)
        }
    }

    private fun decrypt(encoded: String): String {
        require(encoded.length <= maximumEncryptedRecordCharacters)
        val record = Base64.decode(encoded, Base64.NO_WRAP)
        try {
            require(record.size >= 2 + minimumInitializationVectorBytes + authenticationTagBytes)
            require(record[0] == recordVersion)
            val initializationVectorLength = record[1].toInt() and 0xff
            require(initializationVectorLength in minimumInitializationVectorBytes..maximumInitializationVectorBytes)
            val ciphertextOffset = 2 + initializationVectorLength
            require(record.size >= ciphertextOffset + authenticationTagBytes)

            val initializationVector = record.copyOfRange(2, ciphertextOffset)
            val ciphertext = record.copyOfRange(ciphertextOffset, record.size)
            val plaintext: ByteArray
            try {
                val cipher = Cipher.getInstance(cipherTransformation)
                cipher.init(
                    Cipher.DECRYPT_MODE,
                    getExistingKey(),
                    javax.crypto.spec.GCMParameterSpec(authenticationTagBits, initializationVector),
                )
                plaintext = cipher.doFinal(ciphertext)
            } finally {
                initializationVector.fill(0)
                ciphertext.fill(0)
            }
            try {
                require(plaintext.size <= maximumRecordBytes)
                return Charsets.UTF_8
                    .newDecoder()
                    .onMalformedInput(CodingErrorAction.REPORT)
                    .onUnmappableCharacter(CodingErrorAction.REPORT)
                    .decode(ByteBuffer.wrap(plaintext))
                    .toString()
            } finally {
                plaintext.fill(0)
            }
        } finally {
            record.fill(0)
        }
    }

    private fun getOrCreateKey(): SecretKey {
        val keyStore = KeyStore.getInstance(androidKeyStore).apply { load(null) }
        existingKey(keyStore)?.let {
            return it
        }

        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, androidKeyStore)
        generator.init(
            KeyGenParameterSpec.Builder(
                keyAlias,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .setRandomizedEncryptionRequired(true)
                .build(),
        )
        return generator.generateKey()
    }

    private fun getExistingKey(): SecretKey {
        val keyStore = KeyStore.getInstance(androidKeyStore).apply { load(null) }
        return existingKey(keyStore)
            ?: throw IllegalStateException("identity storage key is unavailable")
    }

    private fun existingKey(keyStore: KeyStore): SecretKey? {
        val key = keyStore.getKey(keyAlias, null) ?: return null
        return key as? SecretKey
            ?: throw IllegalStateException("identity storage key has invalid type")
    }

    private companion object {
        const val preferencesName = "coderoam_mobile_identity_v1"
        const val recordKey = "x25519_identity"
        const val keyAlias = "coderoam_mobile_identity_storage_v1"
        const val androidKeyStore = "AndroidKeyStore"
        const val cipherTransformation = "AES/GCM/NoPadding"
        const val maximumRecordBytes = 1024
        const val maximumEncryptedRecordCharacters = 2048
        const val minimumInitializationVectorBytes = 12
        const val maximumInitializationVectorBytes = 16
        const val authenticationTagBytes = 16
        const val authenticationTagBits = authenticationTagBytes * 8
        const val recordVersion: Byte = 1
        val storageLock = Any()
    }
}
