// Package session holds one game in progress: the mode, the score and the
// lives. Games add their own fields.
package session

import "github.com/comprofix/nightcrossing/internal/config"

// Session is one game in progress.
type Session struct {
	Mode  config.Mode
	Score int
	Lives int
}

// New starts a game in a mode with the given lives.
func New(mode config.Mode, lives int) *Session {
	return &Session{Mode: mode, Lives: lives}
}

// AddScore adds points, keeping the score between 0 and config.MaxScore.
func (s *Session) AddScore(points int) {
	s.Score = min(max(s.Score+points, 0), config.MaxScore)
}

// LoseLife removes a life and reports whether none are left.
func (s *Session) LoseLife() (gameOver bool) {
	s.Lives = max(s.Lives-1, 0)
	return s.Lives == 0
}
