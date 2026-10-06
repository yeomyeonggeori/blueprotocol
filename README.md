# blueprotocol

blueprotocol holds the records, tool descriptors and model transport that an agent harness and the host running it both read. Neither side owns them, so each can change without the other.

The module imports neither bluecollar nor blueclaw. `dependency_closure_test.go` fails when any package here reaches either one.
