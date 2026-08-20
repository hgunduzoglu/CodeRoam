import 'dart:ffi';

const int expectedNoiseCoreAbiVersion = 1;

@Native<Uint32 Function()>(
  symbol: 'coderoam_noise_ffi_probe',
  assetId: 'package:coderoam_noise_ffi/coderoam_noise_ffi.dart',
)
external int _nativeNoiseCoreProbe();

int coderoamNoiseCoreAbiVersion() => _nativeNoiseCoreProbe();
