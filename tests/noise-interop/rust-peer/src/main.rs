use std::io::{self, BufRead, Read, Write};

const MAX_HANDSHAKE_MESSAGE_SIZE: usize = 4 * 1024;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let arguments = std::env::args().collect::<Vec<_>>();
    let wrong_psk_bad_message1 = arguments
        .iter()
        .any(|argument| argument == "--wrong-psk-bad-msg1");
    let wrong_psk =
        wrong_psk_bad_message1 || arguments.iter().any(|argument| argument == "--wrong-psk");
    let empty_message3 = arguments.iter().any(|argument| argument == "--empty-msg3");
    let flood_output = arguments
        .iter()
        .any(|argument| argument == "--flood-output");
    let psk_byte = if wrong_psk { 0xa4_u8 } else { 0xa5_u8 };
    // Fixed public test material makes failures reproducible. It is never a production identity.
    let static_private = [0x24_u8; 32];
    let psk = [psk_byte; 32];
    let params = "Noise_XXpsk3_25519_ChaChaPoly_BLAKE2s".parse()?;
    let mut state = snow::Builder::new(params)
        .local_private_key(&static_private)?
        .psk(3, &psk)?
        .prologue(b"coderoam-pairing-v1")?
        .build_initiator()?;

    let mut write_buffer = [0_u8; MAX_HANDSHAKE_MESSAGE_SIZE];
    let message1_size = state.write_message(b"client-m1", &mut write_buffer)?;
    if wrong_psk_bad_message1 {
        println!("msg1=00");
        io::stdout().flush()?;
        return Ok(());
    }
    println!("msg1={}", hex::encode(&write_buffer[..message1_size]));
    io::stdout().flush()?;

    if flood_output {
        loop {
            println!("unexpected=bounded-parser-regression");
            eprintln!("discarded-stderr-regression");
            io::stdout().flush()?;
        }
    }

    let mut message2_text = String::new();
    io::stdin()
        .lock()
        .take((MAX_HANDSHAKE_MESSAGE_SIZE * 2 + 1) as u64)
        .read_line(&mut message2_text)?;
    let message2 = hex::decode(message2_text.trim())?;
    if message2.len() > MAX_HANDSHAKE_MESSAGE_SIZE {
        return Err("message 2 exceeds the handshake bound".into());
    }

    let mut read_buffer = [0_u8; MAX_HANDSHAKE_MESSAGE_SIZE];
    let payload2_size = state.read_message(&message2, &mut read_buffer)?;
    if &read_buffer[..payload2_size] != b"agent-m2" {
        return Err("unexpected responder payload 2".into());
    }

    if empty_message3 {
        println!("msg3=");
    } else {
        let message3_size = state.write_message(b"client-m3", &mut write_buffer)?;
        println!("msg3={}", hex::encode(&write_buffer[..message3_size]));
    }
    println!("binding={}", hex::encode(state.get_handshake_hash()));
    println!(
        "peer={}",
        hex::encode(
            state
                .get_remote_static()
                .ok_or("missing responder static")?
        )
    );
    io::stdout().flush()?;
    Ok(())
}
