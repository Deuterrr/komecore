package slug

type Generator interface {
	Generate(input string) string
}
