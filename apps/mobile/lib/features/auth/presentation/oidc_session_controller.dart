import 'package:coderoam/features/auth/application/oidc_session_lifecycle.dart';
import 'package:flutter/foundation.dart';

enum OidcSessionStatus {
  initial,
  restoring,
  signedOut,
  signingIn,
  signedIn,
  signingOut,
  failed,
}

final class OidcSessionController extends ChangeNotifier {
  OidcSessionController(OidcSession session) : _session = session;

  final OidcSession _session;
  OidcSessionStatus _status = OidcSessionStatus.initial;
  bool _disposed = false;

  OidcSessionStatus get status => _status;

  Future<void> restore() async {
    if (_disposed || _status != OidcSessionStatus.initial) {
      return;
    }
    _status = OidcSessionStatus.restoring;
    notifyListeners();
    try {
      final restored = await _session.restore();
      if (_disposed) {
        return;
      }
      _status =
          restored ? OidcSessionStatus.signedIn : OidcSessionStatus.signedOut;
    } catch (_) {
      if (_disposed) {
        return;
      }
      _status = OidcSessionStatus.failed;
    }
    notifyListeners();
  }

  Future<void> signIn() async {
    if (_disposed ||
        (_status != OidcSessionStatus.signedOut &&
            _status != OidcSessionStatus.failed)) {
      return;
    }
    _status = OidcSessionStatus.signingIn;
    notifyListeners();
    try {
      final signedIn = await _session.signIn();
      if (_disposed) {
        return;
      }
      _status =
          signedIn ? OidcSessionStatus.signedIn : OidcSessionStatus.signedOut;
    } catch (_) {
      if (_disposed) {
        return;
      }
      _status = OidcSessionStatus.failed;
    }
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
