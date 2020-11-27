package webdav

/*
	IN THIS FILE: user management
		- load user from db
		- create new user
		- configure access
*/

import (
	"bufio"
	impl "github.com/SchnorcherSepp/storage/defaultimpl"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// _Users manages all users.
type _Users struct {
	debug         bool
	userFile      string
	userFileMTime int64

	mux            *sync.Mutex
	lastUpdate     time.Time
	updateInterval time.Duration
	users          map[string]*_User
}

// initUsers return the user management.
// The user file is monitored for changes (ModTime).
//
//  line format: <username>:<bcrypt hash>:<access prefix>:...
//    * one user per line
//    * comment lines starts with '#' or ';' or '/' or '%'
//    * there can be several access prefix, but at least one (separator is ':')
//    * do not use ' ' in your username or access prefix
func initUsers(userFile string, debugLvl uint8) *_Users {
	// debug (0=off, 1=debug, 2=high)
	debug := debugLvl >= impl.DebugLow

	// new users
	us := &_Users{
		debug:         debug,
		userFile:      userFile,
		userFileMTime: -999999999999999999,

		mux:            new(sync.Mutex),
		lastUpdate:     time.Time{},
		updateInterval: 15 * time.Second, // default: 15 seconds
		users:          make(map[string]*_User),
	}

	// first update and return
	us.userUpdate()
	return us
}

// Get return a user and call update()
func (us *_Users) Get(username string) (*_User, bool) {

	// lock/unlock
	us.mux.Lock() // LOCK
	defer us.mux.Unlock()

	// call update
	// (it only has an effect every n seconds)
	us.userUpdate()

	// get and return user
	user, ok := us.users[username]
	return user, ok
}

// userUpdate updates the internal user list of passwords and permissions.
// It only has an effect every n seconds.
func (us *_Users) userUpdate() int {

	// only has an effect every n seconds
	if us.lastUpdate.Add(us.updateInterval).After(time.Now()) {
		return -1 // EXIT: too early
	} else {
		us.lastUpdate = time.Now() // set last update
	}

	// only has an effect on new files
	info, err := os.Stat(us.userFile)
	if err != nil {
		log.Printf("WARNING: %s/userUpdate: stat error: %v", packageName, err)
		return -2 // EXIT: file not found
	}
	if info.ModTime().Unix() == us.userFileMTime {
		return -3 // EXIT: file not changed
	}

	//-------------------------------------------------

	// debug log
	if us.debug {
		log.Printf("DEBUG: %s/userUpdate: read user file: '%s'", packageName, us.userFile)
	}

	// load users from file
	newUsers := make(map[string]*_User)
	{
		// open file
		fh, err := os.Open(us.userFile)
		if err != nil {
			log.Printf("WARNING: %s/userUpdate: open error: %v", packageName, err)
		}
		defer fh.Close()

		// read first 1000 lines
		r := bufio.NewReader(fh)
		for i := 0; i < 1000; i++ {
			// read line
			line, _ := r.ReadString('\n')
			line = strings.TrimSpace(line)
			line = strings.ReplaceAll(line, " ", "")
			line = strings.ReplaceAll(line, "\t", "")

			// skip empty lines
			if len(line) == 0 {
				continue // empty
			}

			// skip comments
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") ||
				strings.HasPrefix(line, "/") || strings.HasPrefix(line, "%") {
				continue // comment
			}

			// parse line
			args := strings.Split(line, ":")
			if len(args) < 3 || len(args[0]) == 0 || len(args[1]) == 0 || len(args[2]) == 0 {
				log.Printf("WARNING: %s/userUpdate: invalid line[%d]: %s", packageName, i+1, line)
				continue // invalid
			}

			// add user
			newUser := &_User{
				Username:   args[0],
				PassHash:   args[1],
				pathPrefix: args[2:],
			}
			newUsers[newUser.Username] = newUser

			// debug log
			if us.debug {
				log.Printf("DEBUG: %s/userUpdate: add user '%s' with access prefix %#v", packageName, newUser.Username, newUser.pathPrefix)
			}
		}
	}

	// FINAL: set users and times
	us.users = newUsers
	us.userFileMTime = info.ModTime().Unix()
	return len(us.users)
}

//--------------------------------------------------------------------------------------------------------------------//

// _User contains the settings of a user.
type _User struct {
	Username   string
	PassHash   string
	pathPrefix []string
}

// Allowed checks if the user has permission to access a directory/file
func (u *_User) Allowed(url string) bool {

	// allow root
	if url == "" || url == "/" || url == "." {
		return true
	}

	// allow pathPrefix
	for _, pp := range u.pathPrefix {
		if strings.HasPrefix(url, pp) {
			return true
		}
	}

	// default: false
	return false
}
