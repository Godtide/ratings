package config

//Properties Configuration properties based on env variables.
type Properties struct {
	Port                  string `env:"MY_APP_PORT" env-default:"8080"`
	Host                  string `env:"HOST" env-default:"localhost"`
	DBHost                string `env:"DB_HOST" env-default:"localhost"`
	DBPort                string `env:"DB_PORT" env-default:"27017"`
	DBName                string `env:"DB_NAME" env-default:"rating"`
	UsersCollection       string `env:"USERS_COL_NAME" env-default:"users"`
	UsersRewardCollection string `env:"USERS_REWARD_COL_NAME" env-default:"users_reward"`
	RewardCollection      string `env:"REWARD_COL_NAME" env-default:"reward"`
	WalletCollection      string `env:"USERS_COL_NAME" env-default:"wallet"`
	MasterPrivateKey      string `env:"MASTER_PRIVATE_KEY" env-default:""`
	MasterPublicKey       string `env:"MASTER_PUBLIC_KEY" env-default:""`
	ApiKey                string `env:"ApiKey" env-default:"07f0dfde071243bdbc4c3a53562536cf"`
	ContractAdrress       string `env:"ContractAddress" env-default:"0xB318E25681c0B51DfFA80535Ea49b340c72cC40e"`
}
