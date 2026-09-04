// Package security содержит адаптеры для работы с секретами identity-сервиса.
package security

import "golang.org/x/crypto/bcrypt"

// BcryptHasher хеширует пароли алгоритмом bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher создаёт хешер с указанной стоимостью bcrypt.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash возвращает bcrypt-хеш переданного пароля.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(hash), err
}

// Compare проверяет пароль против сохраненного bcrypt-хеша.
func (h *BcryptHasher) Compare(password string, passwordHash string) error {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
}
