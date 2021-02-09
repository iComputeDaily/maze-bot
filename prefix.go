package main

import "database/sql"
import "strconv"
import "go.uber.org/zap"
import "github.com/andersfylling/disgord"
import "context"
import "strings"
import "errors"
import "unicode/utf8"
import "golang.org/x/text/unicode/norm"
import "unicode"

// Get prefix gets the prefix using the given object with the QueryRow method
func getPrefix(guildID disgord.Snowflake, localDB interface{
	QueryRow(query string, args ...interface{}) *sql.Row
}) string {
	// Has the message prefix
	var prefix string

	// Get prefix from database
	row := localDB.QueryRow("SELECT prefix FROM prefixes WHERE guild_id = $1",
		strconv.FormatUint(uint64(guildID), 10))
	err := row.Scan(&prefix)

	if err != nil {
		switch err {
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

func setPrefix(msg *disgord.Message, prefix string) (string, error) {
	// Get arguments from message
	args := strings.Split(msg.Content, " ")

	// Store the prefix argument
	var newPrefix string

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
				return "", errors.New("Prefix too long.")
			}
			
			char, _ := utf8.DecodeRuneInString(arg)
			if !unicode.IsOneOf(unicode.PrintRanges, char) {
				return "", errors.New("Prefix must be a letter, number, symbol, puctuation charachter, or mark.")
			}
			
			newPrefix = arg
		}
	}
	
	// Start a transaction to hopefully avoid conflicts*double crosses fingers*
	tx, err := stuff.db.BeginTx(context.Background(),
	&sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})
	
	// Check the current prefix
	oldPrefix := getPrefix(msg.GuildID, tx)

	// Bug: Serialization failure not handled
	switch {
	// Same prefix 
	case newPrefix == oldPrefix:
		return oldPrefix, nil

	// Prefix is !
	case newPrefix == stuff.config.General.Prefix:
		_, err = tx.Exec("DELETE FROM prefixes WHERE guild_id = $1;",
		strconv.FormatUint(uint64(msg.GuildID), 10))

	// No Prefix
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
		return "", errors.New(stuff.config.Messages.GenericError)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return "", errors.New(stuff.config.Messages.GenericError)
	}

	return newPrefix, nil
}
