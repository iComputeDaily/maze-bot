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
			stuff.logger.Error("Failed to retrive database results!",
				zap.Uint64("guild_id", uint64(guildID)),
				zap.Error(err))
		}
	}

	return prefix
}

func setPrefixTx(msg *disgord.Message, newPrefix string) error {
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
			stuff.logger.Error("Failed to rollback transaction",
				zap.NamedError("dbError", err), zap.NamedError("rollbackError", rollbackErr))
		default:
			stuff.logger.Error("DB update failed, transaction rolled back sucsessfully.",
				zap.Error(err))
		}

		return err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		stuff.logger.Error("Failed to commit transaction.", zap.Error(err))
		return err
	}
	return nil
}

func setPrefix(msg *disgord.Message, prefix string) (newPrefix string, err error) {
	if msg.IsDirectMessage() {
		return "", errors.New(stuff.config.Messages.NewPrefixInDmError)
	}

	// Get arguments from message
	args := strings.Split(msg.Content, " ")

	// Check all the arguments
	for i, arg := range args {
		switch {
		case arg == "":
			// Do nothing
			break

		case i >= 2:
			// Substitute values and return error
			tooManyArgsError := strings.ReplaceAll(stuff.config.Messages.TooManyArgsError, "<prefix>", prefix)
			return "", errors.New(tooManyArgsError)

		default:
			arg = norm.NFC.String(arg)

			if utf8.RuneCountInString(arg) > 1 {
				return "", errors.New(stuff.config.Messages.PrefixLegnthError)
			}

			char, _ := utf8.DecodeRuneInString(arg)
			if !unicode.In(char, unicode.L, unicode.N, unicode.P, unicode.S) {
				return "", errors.New(stuff.config.Messages.PrefixTypeError)
			}

			newPrefix = arg
		}
	}

	// Try to set the prefix 3 times with backoff
	for i := 0; i < 3; i++ {
		err = setPrefixTx(msg, newPrefix)
		if err == nil || i == 2 {
			break
		}
		time.Sleep(time.Duration(i) * time.Second)
	}

	if err != nil {
		return "", errors.New(stuff.config.Messages.GenericError)
	}

	return newPrefix, nil
}
