package config


const(
	EXTERNAL_LISTEN_PORT = "8080"
	INTERNAL_LISTEN_PORT = "10000"
	PACK_DATA_SIZE_TYPE_LEN = 4	//sizeof(int32)
	MAX_INBOUND_PACK_DATA_SIZE = 1 << 14	//16KB
	MAX_OUTBOUND_PACK_DATA_SIZE = 1 << 20	//1MB
	RING_BUFFER_SIZE = 2 * MAX_INBOUND_PACK_DATA_SIZE
	MAX_TCP_CONN = 100000
	TCP_READ_TIMEOUT = 600	//sec
	TCP_WRITE_TIMEOUT = 300	//sec
	WRITE_PACK_SYNC_CHAN_SIZE = 10
)

type SocketIdType int64