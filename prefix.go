package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/andersfylling/disgord"
	"go.uber.org/zap"
	"golang.org/x/text/unicode/norm"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Get prefix gets the prefix using the given object with the QueryRow method
func getPrefix(guildID disgord.Snowflake, localDB interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}) (prefix string) {
	// Get prefix from database
	row := localDB.QueryRow("SELECT prefix FROM prefixes WHERE guild_id = $1",
		strconv.FormatUint(uint64(guildID), 10))
	err := row.Scan(&prefix)

	// Note that it should never have serialization errors because isolation level is read commited.
	if err != nil {
		switch err {
		// Set default prefix if none is stored
		case sql.ErrNoRows:
			prefix = stuff.config.General.Prefix
		default:
			prefix = stuff.config.General.Prefix
		}
	}

	return prefix
}

func setPrefixTx(msg *disgord.Message, newPrefix string, logger *zap.SugaredLogger) error {
	// Start a transaction to hopefully avoid conflicts*double crosses fingers*
	tx, err := stuff.db.BeginTx(context.Background(),
		&sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})

	// Check the current prefix
	oldPrefix := getPrefix(msg.GuildID, tx)

	// Bug: Serialization failure not handled
	switch {
	// Same prefix
	case newPrefix == oldPrefix:
		return nil

	// New prefix is default
	case newPrefix == stuff.config.General.Prefix:
		_, err = tx.Exec("DELETE FROM prefixes WHERE guild_id = $1;",
			strconv.FormatUint(uint64(msg.GuildID), 10))

	// Current prefix is default
	case oldPrefix == stuff.config.General.Prefix:
		_, err = tx.Exec("INSERT INTO prefixes (guild_id, prefix) VALUES ($1, $2);",
			strconv.FormatUint(uint64(msg.GuildID), 10), newPrefix)

	// Otherwise
	default:
		_, err = tx.Exec("UPDATE prefixes SET prefix = $2 WHERE guild_id = $1;",
			strconv.FormatUint(uint64(msg.GuildID), 10), newPrefix)
	}
	// Handle errors
	if err != nil {
		rollbackErr := tx.Rollback()
		switch {
		case rollbackErr != nil:
			logger.Errorw("Failed to rollback transaction",
				"dbError", err,
				"rollbackError", rollbackErr,
			)
		default:
			logger.Errorw("DB update failed, transaction rolled back sucsessfully.", "error", err)
		}

		return err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		logger.Errorw("Failed to commit transaction.", "error", err)
		return err
	}
	return nil
}

func setPrefix(msg *disgord.Message, prefix string, logger *zap.SugaredLogger) (newPrefix string, err error) {
	// Get arguments from message
	args := strings.Split(msg.Content, " ")

	// Check all the arguments
	for i, arg := range args {
		switch {
		case arg == "":
			// Do nothing
			break

		case i >= 2:
			logger.Infow("Command has too many arguments")
			// Reply with error
			tooManyArgsError := strings.ReplaceAll(stuff.config.Messages.TooManyArgsError, "<prefix>", prefix)
			return "", errors.New(tooManyArgsError)

		default:
			arg = norm.NFC.String(arg)

			if utf8.RuneCountInString(arg) > 1 {
				logger.Infow("Prefix has too many charachters", "Argument", arg)
				return "", errors.New(stuff.config.Messages.PrefixLegnthError)
			}

			char, _ := utf8.DecodeRuneInString(arg)
			if !unicode.In(char, unicode.L, unicode.N, unicode.P, unicode.S) {
				logger.Infow("Prefix charachter is not of allowed type", "Argument", arg)
				return "", errors.New(stuff.config.Messages.PrefixTypeError)
			}

			newPrefix = arg
			logger = logger.With("NewPrefix", newPrefix)
		}
	}

	// Try to set the prefix 3 times with backoff
	for i := 0; i < 3; i++ {
		err = setPrefixTx(msg, newPrefix, logger)
		if err == nil || i == 2 {
			break
		}
		logger.Warnw("Failed atempt to set prefix",
			"error", err,
			"AttemptNum", i+1,
		)
		time.Sleep(time.Duration(i) * time.Second)
	}

	if err != nil {
		logger.Errorw("Failed to Set the prefix", "error", err)
		return "", errors.New(stuff.config.Messages.GenericError)
	}

	return newPrefix, nil
}
