package dumbo

// Store intergace
type Store interface {
	Save(s *Session) error
	Delete(id string) error
	Read(id string) (*Session, error)
}
