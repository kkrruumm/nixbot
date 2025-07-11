package main

import (
	"github.com/bwmarrin/discordgo"

	"log"
	"os"
	"os/signal"
	"os/exec"
	"strings"
	"syscall"
	"io"
	"slices"
	"math/rand"
	"strconv"
)

// feel free to tweak these if you're forking
// for your own server or something
const prefix string = "!nb" // prefix for all commands, bot will ignore any message that doesnt begin with this
const botbanrole string = "1372358865412292689" // if user has this role ID, the bot will ignore any command they send

// modify this to filter the allowed cows
// by default this just corresponds to the cows that void linux
// cowsay provides by default, with "default" removed because it's redundant
var available_cows = []string{"beavis.zen", "blowfish", "bong", "bud-frogs", "bunny", "cheese", "cower", "daemon", "dragon",
"dragon-and-cow", "elephant", "elephant-in-snake", "eyes", "flaming-sheep", "ghostbusters",
"head-in", "hellokitty", "kiss", "kitty", "koala", "kosh", "luke-koala", "meow", "milk", "moofasa", "moose",
"mutilated", "ren", "satanic", "sheep", "skeleton", "small", "stegosaurus", "stimpy", "supermilker",
"surgery", "three-eyes", "turkey", "turtle", "tux", "udder", "vader", "vader-koala", "www"}

func checkInput(r []rune) bool {
	// used by cowsay and figlet to restrict codeblocks primarily
	// so that their codeblock doesn't get an early termination by a user
	// in order to ping people or otherwise via the bot
	// can append more runes to this to restrict more characters
	var disallowed_characters = []rune{'`'}

	for _, rune_value := range r {
		if slices.Contains(disallowed_characters, rune_value) {
			return false
		}
	}
	return true
}

func main() {
	// i'm watching you
	// this bot is intended to be run as a service, typically with its HOME being set to /var/lib/nixbot
	// both the executable and this logfile should be stored there
	homedir := os.Getenv("HOME")
	f, err := os.OpenFile(homedir + "/nixbot.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("nixbot: cannot open log file '", err, "'")
	}
	defer f.Close()

	// should continue printing to stdout incase bot is being run
	// via tmux or something like that
	split_log := io.MultiWriter(os.Stdout, f)
	log.SetOutput(split_log)

	// set this via .profile for the bot user
	token := os.Getenv("NIXBOT_TOKEN")
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}

	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		opts := strings.Split(m.Content, " ")

		if opts[0] != prefix {
			return
		}

		if len(opts) < 2 {
			s.ChannelMessageSend(m.ChannelID, "not enough arguments:\nexpected ``!nb <option>``")
			return
		}

		if slices.Contains(m.Member.Roles, botbanrole) {
			log.Print("nixbot: botbanned user '", m.Author.Username, m.Author.ID, "' attempted to run command ", opts[1:])
			return
		}

		switch command := opts[1]; command {
		case "fortune":
			fortune_output := exec.Command("/usr/sbin/fortune")
			stdout, err := fortune_output.Output()

			if err != nil {
				log.Fatal(err)
			}

			s.ChannelMessageSend(m.ChannelID, "```" + string(stdout) + "```")

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran command [fortune]")

		case "cowsay":
			opt_runes := []rune(strings.Join(opts[2:], " "))
			if !checkInput(opt_runes) {
				s.ChannelMessageSend(m.ChannelID, "illegal character in prompt")
				return
			}

			// make sure we actually have enough opts so the bot doesn't crash
			if len(opts) < 3 {
				s.ChannelMessageSend(m.ChannelID, "not enough arguments:\nexpected ``!nb cowsay <phrase>`` or ``nb cowsay --<cow> <phrase>``")
				return
			}

			var cow_style string
			if strings.Contains(opts[2], "--") {
				if len(opts) < 4 {
					s.ChannelMessageSend(m.ChannelID, "not enough arguments:\nexpected ``!nb cowsay <phrase>`` or ``nb cowsay --<cow> <phrase>``")
					return
				}
				cow_style = strings.ReplaceAll(opts[2], "--", "")
				if !slices.Contains(available_cows, cow_style) {
					s.ChannelMessageSend(m.ChannelID, "invalid cow: `" + cow_style + "`\nsee ``!nb cows`` for a list of valid cows")
					return
				}
			} else {
				cow_style = "default"
			}

			// using stdin to send cowsay the prompt should get rid of
			// any chance of shell injection
			cmd := exec.Command("/usr/sbin/cowsay", "-f", cow_style)
			if cow_style == "default" {
				string := strings.Join(opts[2:], " ")
				cmd.Stdin = strings.NewReader(string)
			} else {
				string := strings.Join(opts[3:], " ")
				cmd.Stdin = strings.NewReader(string)
			}
			var out strings.Builder
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				log.Print(err)
			}

			s.ChannelMessageSend(m.ChannelID, "```" + out.String() + "```")

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran command 'cowsay ", opts[2:], "'")

		case "cows":
			cows_output := strings.Join(available_cows[:], " ")
			s.ChannelMessageSend(m.ChannelID, "available cows:```" + cows_output + "```")

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran command [cows]")

		case "figlet":
			if len(opts) < 3 {
				s.ChannelMessageSend(m.ChannelID, "not enough arguments:\nexpected ``!nb figlet <phrase>``")
				return
			}

			opt_runes := []rune(strings.Join(opts[2:], " "))
			if !checkInput(opt_runes) {
				s.ChannelMessageSend(m.ChannelID, "illegal character in prompt")
				return
			}

			cmd := exec.Command("/usr/sbin/figlet")
			string := strings.Join(opts[2:], " ")
			cmd.Stdin = strings.NewReader(string)
			var out strings.Builder
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				log.Print(err)
			}

			s.ChannelMessageSend(m.ChannelID, "```" + out.String() + "```")

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran command 'figlet ", opts[2:], "'")

		case "greentext":
			greentext_output := strings.Join(opts[2:], " ")
			s.ChannelMessageSend(m.ChannelID, "```ansi\n\u001b[1;0;32m>" + greentext_output + "```")

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran command 'greentext ", opts[2:], "'")

		case "me":
			me_output := strings.Join(opts[2:], " ")
			s.ChannelMessageSend(m.ChannelID, "<@" + m.Author.ID + "> " + me_output)
			s.ChannelMessageDelete(m.ChannelID, m.ID)

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran command 'me ", opts[2:], "'")

		case "help":
			var output_strings = []string{"- ``!nb fortune`` - outputs out a random fortune",
			"- ``!nb cowsay <phrase>`` - outputs cowsay with ``<phrase>``",
			"- ``!nb cowsay --<cow> <phrase>`` - same as regular cowsay, but you get to choose a cow",
			"- ``!nb cows`` - outputs list of valid cows",
			"- ``!nb figlet <phrase>`` - outputs figlet with ``<phrase>``",
			"- ``!nb greentext <phrase>`` - outputs a greentext with ``<phrase>``",
			"- ``!nb me <phrase>`` - outputs an IRC-esque /me with ``<phrase>``",
			"- ``!nb avatar`` - returns URL of user avatar",
			"- ``!nb xkcd`` - returns URL of a random xkcd comic"}

			help_output := strings.Join(output_strings[:], "\n")
			s.ChannelMessageSend(m.ChannelID, help_output)

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran command [help]")

		case "avatar":
			avatar_output := m.Author.AvatarURL("")
			s.ChannelMessageSend(m.ChannelID, avatar_output)

		case "xkcd":
			xkcd_number := strconv.Itoa(rand.Intn(3092 - 1))
			xkcd_output := "https://xkcd.com/" + xkcd_number

			s.ChannelMessageSend(m.ChannelID, xkcd_output)

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran command [xkcd]")

		default:
			s.ChannelMessageSend(m.ChannelID, "invalid option:\nsee ``!nb help`` for valid options")

			log.Print("nixbot: user '", m.Author.Username, ":", m.Author.ID, "' ran invalid command ", opts[1:])
		}
	})

	session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = session.Open()
	if err != nil {
		log.Fatal(err)
	}

	defer session.Close()

	log.Print("nixbot: i'm alive!!!")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	log.Print("nixbot: bravo 6 going dark")
}
