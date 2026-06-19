# ACL Verifier

The verifier module checks assumptions before ACL tries to launch or patch anything.
It is intended to answer simple questions such as:

- Does the runtime directory exist?
- Are the expected glibc-style libraries present?
- Does this file look like an ELF binary?
- Does the copied runtime still look relocatable, or is it tied to Termux paths?

The v0.1 verifier is intentionally conservative and reports missing runtime pieces plainly.
