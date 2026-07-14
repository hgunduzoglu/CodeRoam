# CodeRoam Relay Guidance

- Relay is the normal runtime data path from the first integrated data-plane milestone.
- Accept client and outbound agent TLS WebSockets.
- Validate short-lived tickets, pair endpoints, and forward opaque encrypted frames.
- Never decrypt, parse, persist, or log application payloads.
- Redis contains presence/routing/rate-limit metadata only.
- Bound every frame, queue, replay marker, connection, and lifetime.
- Prioritize control, terminal input, and editor acknowledgements over bulk streams.
- Test slow consumers, reconnect storms, invalid/replayed tickets, drain, and abrupt disconnect.
