# Protobuf contracts

Put only justified internal gRPC contracts in this directory. HTTP/JSON is the
default transport; add a `.proto` file when a service needs a typed RPC or
streaming contract. Run `task generate:proto` after adding or changing one.
