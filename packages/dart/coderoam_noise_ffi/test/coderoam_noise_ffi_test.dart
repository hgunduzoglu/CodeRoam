import 'package:coderoam_noise_ffi/coderoam_noise_ffi.dart';
import 'package:test/test.dart';

void main() {
  test('bundled Rust core supports the approved XXpsk3 suite', () {
    expect(coderoamNoiseCoreAbiVersion(), expectedNoiseCoreAbiVersion);
  });
}
