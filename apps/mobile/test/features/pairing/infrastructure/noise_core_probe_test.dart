import 'package:coderoam_noise_ffi/coderoam_noise_ffi.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('mobile package resolves the bundled Noise core ABI', () {
    expect(coderoamNoiseCoreAbiVersion(), expectedNoiseCoreAbiVersion);
  });
}
