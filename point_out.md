#### Why is the hospital scope read only from the JWT, never from the request body?

Anything the client sends is untrusted input. If authorization logic ever reads a hospital field from the request, an attacker just changes that field and reads another hospital's patients. The JWT claim is set server-side at login and can't be forged without the signing key, so it's the only thing authorization is allowed to trust.
