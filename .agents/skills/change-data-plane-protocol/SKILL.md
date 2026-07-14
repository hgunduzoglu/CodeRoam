---
name: change-data-plane-protocol
description: Safely evolve CodeRoam Protobuf, pairing, Noise session, multiplexing, relay, reconnect, resume, or channel semantics.
---

# Change the Data-Plane Protocol

1. Read protocol, relay, agent, mobile, and security guidance.
2. Map every producer and consumer.
3. State compatibility and rollout behavior.
4. Define ordering, ACK, cancellation, deadline, size, flow control, priority, replay, resume, and
   malformed-input behavior.
5. Preserve relay opacity and E2E encryption.
6. Initial pairing remains XXpsk3 with a high-entropy secret; paired sessions remain IK unless an
   approved ADR changes this.
7. Never reuse Protobuf field numbers or hand-edit generated output.
8. Run:

```bash
buf lint
buf breaking --against '.git#branch=main'
make proto
```

9. Add mixed-version, malformed-frame, replay, reconnect, slow-consumer, and size-limit tests.
10. Run all affected Go/Dart consumers and the final gate.
