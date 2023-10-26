package config

//Properties Configuration properties based on env variables.
type Properties struct {
	Port                  string `env:"MY_APP_PORT" env-default:"8080"`
	Host                  string `env:"HOST" env-default:"localhost"`
	DBHost                string `env:"DB_HOST" env-default:"localhost"`
	DBPort                string `env:"DB_PORT" env-default:"27017"`
	DBName                string `env:"DB_NAME" env-default:"tronics"`
	ProductCollection     string `env:"PRODUCTS_COL_NAME" env-default:"products"`
	UsersCollection       string `env:"USERS_COL_NAME" env-default:"users"`
	UsersRewardCollection string `env:"USERS_REWARD_COL_NAME" env-default:"users_reward"`
	RewardCollection      string `env:"REWARD_COL_NAME" env-default:"reward"`
	WalletCollection      string `env:"USERS_COL_NAME" env-default:"wallet"`
	JwtTokenSecret        string `env:"JWT_TOKEN_SECRET" env-default:"abrakadabra"`
	MasterPrivateKey      string `env:"MASTER_PRIVATE_KEY" env-default:""`
	MasterPublicKey       string `env:"MASTER_PUBLIC_KEY" env-default:""`
}
