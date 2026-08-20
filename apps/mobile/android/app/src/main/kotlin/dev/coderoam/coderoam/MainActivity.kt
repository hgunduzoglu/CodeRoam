package dev.coderoam.coderoam

import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        val storage = MobileIdentityStorage(applicationContext)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName)
            .setMethodCallHandler { call, result ->
                try {
                    when (call.method) {
                        "read" -> result.success(storage.read())
                        "createIfAbsent" -> {
                            val value = call.argument<String>("value")
                                ?: throw IllegalArgumentException("missing identity value")
                            result.success(storage.createIfAbsent(value))
                        }
                        else -> result.notImplemented()
                    }
                } catch (_: Exception) {
                    result.error(
                        "identity_storage_unavailable",
                        "Identity storage is unavailable.",
                        null,
                    )
                }
            }
    }

    private companion object {
        const val channelName = "dev.coderoam/mobile_identity_storage_v1"
    }
}
