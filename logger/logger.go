// Copyright (C) 2016-2017 vLife Systems Ltd <http://vlifesystems.com>
// Licensed under an MIT licence.  Please see LICENSE.md for details.

package logger

import (
	"github.com/kardianos/service"
	"github.com/vlifesystems/rulehunter/quitter"
)

type Logger interface {
	Run(*quitter.Quitter)
	Info(string)
	Error(error) error
	SetSvcLogger(service.Logger)
}

type SvcLogger struct {
	svcLogger service.Logger
}

func NewSvcLogger() *SvcLogger {
	return &SvcLogger{
		svcLogger: nil,
	}
}

func (l *SvcLogger) Run(quit *quitter.Quitter) {
	if l.svcLogger == nil {
		panic("service logger not set")
	}
	quit.Add()
	defer quit.Done()

	for {
		select {
		case <-quit.C:
			return
		}
	}
}

func (l *SvcLogger) SetSvcLogger(logger service.Logger) {
	l.svcLogger = logger
}

// Error logs an error and returns the same error
func (l *SvcLogger) Error(err error) error {
	l.svcLogger.Error(err.Error())
	return err
}

func (l *SvcLogger) Info(msg string) {
	l.svcLogger.Info(msg)
}
