// Copyright (C) 2016-2017 vLife Systems Ltd <http://vlifesystems.com>
// Licensed under an MIT licence.  Please see LICENSE.md for details.

package testhelpers

import (
	"github.com/kardianos/service"
	"github.com/vlifesystems/rulehunter/quitter"
)

type Logger struct {
	entries   []Entry
	isRunning bool
}

type Entry struct {
	Level Level
	Msg   string
}

type Level int

const (
	Info Level = iota
	Error
)

func NewLogger() *Logger {
	return &Logger{
		entries:   make([]Entry, 0),
		isRunning: false,
	}
}

func (l *Logger) Run(quit *quitter.Quitter) {
	quit.Add()
	defer quit.Done()
	l.isRunning = true
	defer func() { l.isRunning = false }()
	for {
		select {
		case <-quit.C:
			return
		}
	}
}

// Running returns whether Logger is running
func (l *Logger) Running() bool {
	return l.isRunning
}

func (l *Logger) SetSvcLogger(logger service.Logger) {
}

func (l *Logger) Error(err error) error {
	entry := Entry{
		Level: Error,
		Msg:   err.Error(),
	}
	l.entries = append(l.entries, entry)
	return err
}

func (l *Logger) Info(msg string) {
	entry := Entry{
		Level: Info,
		Msg:   msg,
	}
	l.entries = append(l.entries, entry)
}

func (l *Logger) GetEntries() []Entry {
	return l.entries
}
