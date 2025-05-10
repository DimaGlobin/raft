package config

type Config struct {
	Id       string `json:"id" env:"ID"`
	Port     string `json:"port" env:"PORT"`
	Self     string `json:"self" env:"SELF"` // http://localhost:8080
	IsLeader bool   `json:"is_leader" env:"IS_LEADER"`
}

// func Load() Config {
// 	return Config{
// 		IsLeader: os.Getenv("IS_LEADER") == "true",
// 		Port:     os.Getenv("PORT"),
// 		Replicas: strings.Split(os.Getenv("SLAVES"), ","), //TODO: переделать
// 	}
// }
