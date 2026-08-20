use snow::params::NoiseParams;

const ABI_VERSION: u32 = 1;
const MAX_HANDSHAKE_MESSAGE_SIZE: usize = 4 * 1024;

#[unsafe(no_mangle)]
pub extern "C" fn coderoam_noise_ffi_probe() -> u32 {
    let Ok(params) = "Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s".parse::<NoiseParams>() else {
        return 0;
    };

    // Public deterministic probe material only; production identities never use this path.
    let private_key = [0x24_u8; 32];
    let pairing_secret = [0xa5_u8; 32];
    let Ok(mut state) = snow::Builder::new(params)
        .local_private_key(&private_key)
        .and_then(|builder| builder.psk(3, &pairing_secret))
        .and_then(|builder| builder.prologue(b"coderoam-pairing-v1"))
        .and_then(|builder| builder.build_initiator())
    else {
        return 0;
    };

    let mut message = [0_u8; MAX_HANDSHAKE_MESSAGE_SIZE];
    match state.write_message(b"ffi-probe", &mut message) {
        Ok(size) if size > 0 => ABI_VERSION,
        _ => 0,
    }
}
