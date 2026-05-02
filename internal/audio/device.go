package audio

type Device struct {
	ID      string
	Name    string
	Driver  string
	Default bool
}

type RecordOptions struct {
	Device     string
	Driver     string
	SampleRate int
	Format     string
}
