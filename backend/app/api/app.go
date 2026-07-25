package main

import (
	"time"

	"github.com/pritish-codes/homebot/backend/internal/core/services"
	"github.com/pritish-codes/homebot/backend/internal/core/services/reporting/eventbus"
	"github.com/pritish-codes/homebot/backend/internal/data/ent"
	"github.com/pritish-codes/homebot/backend/internal/data/repo"
	"github.com/pritish-codes/homebot/backend/internal/sys/config"
	"github.com/pritish-codes/homebot/backend/internal/sys/otel"
	"github.com/pritish-codes/homebot/backend/pkgs/mailer"
)

type app struct {
	conf                *config.Config
	mailer              mailer.Mailer
	db                  *ent.Client
	repos               *repo.AllRepos
	services            *services.AllServices
	bus                 *eventbus.EventBus
	authLimiter         *authRateLimiter
	notifierTestLimiter *simpleRateLimiter
	otel                *otel.Provider
}

func new(conf *config.Config) *app {
	s := &app{
		conf: conf,
	}

	s.mailer = mailer.Mailer{
		Host:     s.conf.Mailer.Host,
		Port:     s.conf.Mailer.Port,
		Username: s.conf.Mailer.Username,
		Password: s.conf.Mailer.Password,
		From:     s.conf.Mailer.From,
	}

	s.authLimiter = newAuthRateLimiter(s.conf.Auth.RateLimit)
	s.notifierTestLimiter = newSimpleRateLimiter(10, time.Minute, s.conf.Options.TrustProxy) // 10 requests per minute

	return s
}
