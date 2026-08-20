import Foundation
import Security

enum MobileIdentityStorageError: Error {
  case unavailable
}

final class MobileIdentityStorage {
  func read() throws -> String? {
    var query = baseQuery()
    query[kSecReturnData as String] = true
    query[kSecMatchLimit as String] = kSecMatchLimitOne

    var result: CFTypeRef?
    let status = SecItemCopyMatching(query as CFDictionary, &result)
    if status == errSecItemNotFound {
      return nil
    }
    guard status == errSecSuccess,
          let data = result as? Data,
          data.count <= Self.maximumRecordBytes,
          let value = String(data: data, encoding: .utf8)
    else {
      throw MobileIdentityStorageError.unavailable
    }
    return value
  }

  func createIfAbsent(_ value: String) throws -> Bool {
    guard value.utf16.count <= Self.maximumRecordBytes,
          let data = value.data(using: .utf8),
          data.count <= Self.maximumRecordBytes
    else {
      throw MobileIdentityStorageError.unavailable
    }

    var query = baseQuery()
    query[kSecValueData as String] = data
    query[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
    let status = SecItemAdd(query as CFDictionary, nil)
    if status == errSecDuplicateItem {
      return false
    }
    guard status == errSecSuccess else {
      throw MobileIdentityStorageError.unavailable
    }
    return true
  }

  private func baseQuery() -> [String: Any] {
    [
      kSecClass as String: kSecClassGenericPassword,
      kSecAttrService as String: Self.service,
      kSecAttrAccount as String: Self.account,
      kSecAttrSynchronizable as String: kCFBooleanFalse as Any,
    ]
  }

  private static let service = "dev.coderoam.coderoam.mobile-identity"
  private static let account = "x25519-v1"
  private static let maximumRecordBytes = 1024
}
