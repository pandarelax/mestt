package audio

type Device struct {
	ID      string
	Name    string
	Default bool
}

type RecordOptions struct {
	Device     string
	SampleRate int
	Format     string
}
