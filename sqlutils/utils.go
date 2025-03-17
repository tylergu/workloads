package sqlutils

const (
	CreatePlayerSQL      = "INSERT INTO player (id, coins) VALUES (?, ?)"
	GetPlayerSQL         = "SELECT id, coins FROM player WHERE id = ?"
	GetCountSQL          = "SELECT count(*) FROM player"
	GetPlayerWithLockSQL = GetPlayerSQL + " FOR UPDATE"
	UpdatePlayerSQL      = "UPDATE player set coins = ? WHERE id = ?"
	DropTableSQL         = "DROP TABLE IF EXISTS player"
	CreateTableSQL       = "CREATE TABLE player ( `id` VARCHAR(36), `coins` INTEGER, `goods` INTEGER, PRIMARY KEY (`id`) );"
	TicksPerSecond       = 10
	WindowSize           = 100
)
