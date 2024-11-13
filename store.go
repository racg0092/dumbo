package dumbo

// TODO: IMPLEMENT STORAGE INTERFACE
type Store interface {
	Save() error
	Delete() error
}
