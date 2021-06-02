// Package godotenv is a go port of the ruby dotenv library (https://github.com/bkeepers/dotenv)
//
// Examples/readme can be found on the github page at https://github.com/joho/godotenv
//
// The TL;DR is that you make a .env file that looks something like
//
// 		SOME_ENV_VAR=somevalue
//
// and then in your go code you can call
//
// 		godotenv.Load()
//
// and all the env vars declared in .env will be available through os.Getenv("SOME_ENV_VAR")
package godotenv

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Double quoting dollar will cause var references to be disabled, that's not what we want!
//const doubleQuoteSpecialChars = "\\\n\r\"!$`"
const doubleQuoteSpecialChars = "\\\n\r\"!`"

// Load will read your env file(s) and load them into ENV for this process.
//
// Call this function as close as possible to the start of your program (ideally in main)
//
// If you call Load without any args it will default to loading .env in the current path
//
// You can otherwise tell it which files to load (there can be more than one) like
//
//		godotenv.Load("fileone", "filetwo")
//
// It's important to note that it WILL NOT OVERRIDE an env variable that already exists - consider the .env file to set dev vars or sensible defaults
func Load(filenames ...string) (err error) {
	filenames = filenamesOrDefault(filenames)

	for _, filename := range filenames {
		err = loadFile(filename, false, true)
		if err != nil {
			return // return early on a spazout
		}
	}
	return
}

// Overload will read your env file(s) and load them into ENV for this process.
//
// Call this function as close as possible to the start of your program (ideally in main)
//
// If you call Overload without any args it will default to loading .env in the current path
//
// You can otherwise tell it which files to load (there can be more than one) like
//
//		godotenv.Overload("fileone", "filetwo")
//
// It's important to note this WILL OVERRIDE an env variable that already exists - consider the .env file to forcefilly set all vars.
func Overload(filenames ...string) (err error) {
	filenames = filenamesOrDefault(filenames)

	for _, filename := range filenames {
		err = loadFile(filename, true, true)
		if err != nil {
			return // return early on a spazout
		}
	}
	return
}

func ReadNoExpand(filenames ...string) (envMap *EnvMap, err error) {
	return read(false, filenames...)
}

// Read all env (with same file loading semantics as Load) but return values as
// a map rather than automatically writing values into env
func Read(filenames ...string) (envMap *EnvMap, err error) {
	return read(true, filenames...)
}

func read(expand bool, filenames ...string) (envMap *EnvMap, err error) {
	filenames = filenamesOrDefault(filenames)
	envMap = NewEnvMap()

	for _, filename := range filenames {
		individualEnvMap, individualErr := readFile(filename, expand)

		if individualErr != nil {
			err = individualErr
			return // return early on a spazout
		}
		individualEnvMap.Iter(func(k, v string) { envMap.Set(k, v) })
	}

	return
}

// Parse reads an env file from io.Reader, returning a map of keys and values.
func Parse(r io.Reader, expand bool) (envMap *EnvMap, err error) {
	envMap = NewEnvMap()

	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err = scanner.Err(); err != nil {
		return
	}

	for _, fullLine := range lines {
		if !isIgnoredLine(fullLine) {
			var key, value string
			key, value, err = parseLine(fullLine, envMap, expand)

			if err != nil {
				return
			}
			envMap.Set(key, value)
		}
	}
	return
}

//Unmarshal reads an env file from a string, returning a map of keys and values.
func Unmarshal(str string) (envMap *EnvMap, err error) {
	return Parse(strings.NewReader(str), true)
}

// Exec loads env vars from the specified filenames (empty map falls back to default)
// then executes the cmd specified.
//
// Simply hooks up os.Stdin/err/out to the command and calls Run()
//
// If you want more fine grained control over your command it's recommended
// that you use `Load()` or `Read()` and the `os/exec` package yourself.
func Exec(filenames []string, cmd string, cmdArgs []string) error {
	Load(filenames...)

	command := exec.Command(cmd, cmdArgs...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

// Write serializes the given environment and writes it to a file
func Write(envMap *EnvMap, filename string) error {
	content := Marshal(envMap)
	file, error := os.Create(filename)
	if error != nil {
		return error
	}
	_, err := file.WriteString(content)
	return err
}

// Marshal outputs the given environment as a dotenv-formatted environment file.
// Each line is in the format: KEY="VALUE" where VALUE is backslash-escaped.
func Marshal(envMap *EnvMap) string {
	lines := make([]string, 0, envMap.Len())
	envMap.Iter(func(k, v string) {
		lines = append(lines, fmt.Sprintf(`%s="%s"`, k, doubleQuoteEscape(v)))
	})
	// We are being used to create referencing lines! No more sorting..
	//sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n"
}

func filenamesOrDefault(filenames []string) []string {
	if len(filenames) == 0 {
		return []string{".env"}
	}
	return filenames
}

func loadFile(filename string, overload bool, expand bool) error {
	envMap, err := readFile(filename, expand)
	if err != nil {
		return err
	}

	currentEnv := map[string]bool{}
	rawEnv := os.Environ()
	for _, rawEnvLine := range rawEnv {
		key := strings.Split(rawEnvLine, "=")[0]
		currentEnv[key] = true
	}
	envMap.Iter(func(k, v string) {
		if !currentEnv[k] || overload {
			os.Setenv(k, v)
		}
	})

	return nil
}

func readFile(filename string, expand bool) (envMap *EnvMap, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	return Parse(file, expand)
}

func parseLine(line string, envMap *EnvMap, expand bool) (key string, value string, err error) {
	if len(line) == 0 {
		err = errors.New("zero length string")
		return
	}

	// ditch the comments (but keep quoted hashes)
	if strings.Contains(line, "#") {
		segmentsBetweenHashes := strings.Split(line, "#")
		quotesAreOpen := false
		var segmentsToKeep []string
		for _, segment := range segmentsBetweenHashes {
			if strings.Count(segment, "\"") == 1 || strings.Count(segment, "'") == 1 {
				if quotesAreOpen {
					quotesAreOpen = false
					segmentsToKeep = append(segmentsToKeep, segment)
				} else {
					quotesAreOpen = true
				}
			}

			if len(segmentsToKeep) == 0 || quotesAreOpen {
				segmentsToKeep = append(segmentsToKeep, segment)
			}
		}

		line = strings.Join(segmentsToKeep, "#")
	}

	firstEquals := strings.Index(line, "=")
	firstColon := strings.Index(line, ":")
	splitString := strings.SplitN(line, "=", 2)
	if firstColon != -1 && (firstColon < firstEquals || firstEquals == -1) {
		//this is a yaml-style line
		splitString = strings.SplitN(line, ":", 2)
	}

	if len(splitString) != 2 {
		err = errors.New("Can't separate key from value")
		return
	}

	// Parse the key
	key = splitString[0]
	if strings.HasPrefix(key, "export") {
		key = strings.TrimPrefix(key, "export")
	}
	key = strings.TrimSpace(key)

	re := regexp.MustCompile(`^\s*(?:export\s+)?(.*?)\s*$`)
	key = re.ReplaceAllString(splitString[0], "$1")

	// Parse the value
	value = parseValue(splitString[1], envMap, expand)
	return
}

func parseValue(value string, envMap *EnvMap, expand bool) string {

	// trim
	value = strings.Trim(value, " ")

	// check if we've got quoted values or possible escapes
	if len(value) > 1 {
		rs := regexp.MustCompile(`\A'(.*)'\z`)
		singleQuotes := rs.FindStringSubmatch(value)

		rd := regexp.MustCompile(`\A"(.*)"\z`)
		doubleQuotes := rd.FindStringSubmatch(value)

		if singleQuotes != nil || doubleQuotes != nil {
			// pull the quotes off the edges
			value = value[1 : len(value)-1]
		}

		if doubleQuotes != nil {
			// expand newlines
			escapeRegex := regexp.MustCompile(`\\.`)
			value = escapeRegex.ReplaceAllStringFunc(value, func(match string) string {
				c := strings.TrimPrefix(match, `\`)
				switch c {
				case "n":
					return "\n"
				case "r":
					return "\r"
				default:
					return match
				}
			})
			// unescape characters
			e := regexp.MustCompile(`\\([^$])`)
			value = e.ReplaceAllString(value, "$1")
		}

		if singleQuotes == nil && expand {
			value = expandVariables(value, envMap)
		}
	}

	return value
}

func expandVariables(v string, m *EnvMap) string {
	r := regexp.MustCompile(`(\\)?(\$)(\()?\{?([A-Z0-9_]+)?\}?`)

	return r.ReplaceAllStringFunc(v, func(s string) string {
		submatch := r.FindStringSubmatch(s)

		if submatch == nil {
			return s
		}
		if submatch[1] == "\\" || submatch[2] == "(" {
			return submatch[0][1:]
		} else if submatch[4] != "" {
			if val, ok := m.Get(submatch[4]); ok >= 0 {
				return val
			}
			return os.Getenv(submatch[4])
		}
		return s
	})
}

func isIgnoredLine(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	return len(trimmedLine) == 0 || strings.HasPrefix(trimmedLine, "#")
}

func doubleQuoteEscape(line string) string {
	for _, c := range doubleQuoteSpecialChars {
		toReplace := "\\" + string(c)
		if c == '\n' {
			toReplace = `\n`
		}
		if c == '\r' {
			toReplace = `\r`
		}
		line = strings.Replace(line, string(c), toReplace, -1)
	}
	return line
}
