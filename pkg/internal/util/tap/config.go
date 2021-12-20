package tap

type Config struct {
	Name    string
	Net     string
	MTU     int
	Routes  []string
	Gateway string
}
