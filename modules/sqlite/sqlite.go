package sqlite

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func Init() {
	var err error
	db, err = sql.Open("sqlite3", "./wired.db")
	if err != nil {
		log.Fatal(err)
	}

	initTables()
}

func initTables() {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		discord_id TEXT NOT NULL,
		username TEXT NOT NULL,
		discriminator TEXT NOT NULL,
		avatar TEXT NOT NULL,
		role TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}

	/*_, err = db.Exec(`CREATE TABLE IF NOT EXISTS routes (
		route_id TEXT PRIMARY KEY,
		server_host TEXT NOT NULL,
		server_port TEXT NOT NULL,
		proxy_domain TEXT NOT NULL,
		proxy_port TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}*/
}

func CreateUser(discordId, username, discriminator, avatar, role string) error {
	_, err := db.Exec("INSERT INTO users (discord_id, username, discriminator, avatar, role) VALUES (?, ?, ?, ?, ?)", discordId, username, discriminator, avatar, role)
	return err
}

// unsafe, dont allow key to be user input
func GetUser(key, value string) (string, string, string, string, string, error) {
	var discordId, username, discriminator, avatar, role string
	err := db.QueryRow("SELECT discord_id, username, discriminator, avatar, role FROM users WHERE "+key+" = ?", value).Scan(&discordId, &username, &discriminator, &avatar, &role)
	return discordId, username, discriminator, avatar, role, err
}

type DiscordUser struct {
	DiscordId    string `json:"discord_id"`
	Username     string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar       string `json:"avatar"`
	Role         string `json:"role"`
}

func GetUsers() []DiscordUser {
	rows, err := db.Query("SELECT discord_id, username, discriminator, avatar, role FROM users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	users := []DiscordUser{}
	for rows.Next() {
		var discordId, username, discriminator, avatar, role string
		err := rows.Scan(&discordId, &username, &discriminator, &avatar, &role)
		if err != nil {
			log.Fatal(err)
		}

		users = append(users, DiscordUser{
			DiscordId:    discordId,
			Username:     username,
			Discriminator: discriminator,
			Avatar:       avatar,
			Role:         role,
		})
	}

	return users
}

func ChangeUserRole(discordId, role string) error {
	_, err := db.Exec("UPDATE users SET role = ? WHERE discord_id = ?", role, discordId)
	return err
}

func DeleteUser(key, value string) error {
	_, err := db.Exec("DELETE FROM users WHERE "+key+" = ?", value)
	return err
}

/*func CreateRoute(routeId, serverHost, serverPort, proxyDomain, proxyPort string) error {
	_, err := db.Exec("INSERT INTO routes (route_id, server_host, server_port, proxy_domain, proxy_port) VALUES (?, ?, ?, ?, ?)", routeId, serverHost, serverPort, proxyDomain, proxyPort)
	return err
}*/

func Close() {
	db.Close()
}

func GetDB() *sql.DB {
	return db
}
