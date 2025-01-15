package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"log"
)

var config *Config

func main() {
	log.SetFlags(log.Lshortfile)

	// config

	var err error

	config, err = loadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Discord

	dg, err := discordgo.New(config.DiscordBotToken)
	if err != nil {
		log.Fatalf("Error initializing Discord: %v", err)
	}

	dg.AddHandler(handleUserRoleChange)
	dg.AddHandler(handleUserRemove)

	// Run bot & wait for signals
	dg.Identify.Intents = discordgo.IntentsGuildMembers

	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening Discord onnection: %v", err)
		return
	}

	err = syncUserRoles(dg)
	if err != nil {
		log.Fatalf("Error syncing user roles: %v", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
}

func syncUserRoles(dg *discordgo.Session) error {
	log.Print("Starting Discord user roles sync...")

	for _, guild := range dg.State.Guilds {
		log.Printf("Syncing roles for guild %v", guild.Name)

		members, err := dg.GuildMembers(guild.ID, "", 1000)
		if err != nil {
			return err
		}

		request := RoleUpdateRequest{
			GuildID: guild.ID,
			Users:   []UserRoles{},
		}

		for _, member := range members {
			request.Users = append(request.Users, UserRoles{
				UserID: member.User.ID,
				Roles:  member.Roles,
			})
		}

		err = postRolesToHavenAPI(request)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleUserRoleChange(dg *discordgo.Session, event *discordgo.GuildMemberUpdate) {
	guild, err := dg.Guild(event.GuildID)
	if err != nil {
		log.Printf("Error getting guild: %v", err)
		return
	}

	request := RoleUpdateRequest{
		GuildID: guild.ID,
		Users: []UserRoles{
			{
				UserID: event.User.ID,
				Roles:  event.Roles,
			},
		},
	}

	err = postRolesToHavenAPI(request)
	if err != nil {
		log.Printf("Error posting role update to Haven API: %v", err)
		return
	}
}

func handleUserRemove(dg *discordgo.Session, event *discordgo.GuildMemberRemove) {
	guild, err := dg.Guild(event.GuildID)
	if err != nil {
		log.Printf("Error getting guild: %v", err)
		return
	}

	request := RoleUpdateRequest{
		GuildID: guild.ID,
		Users: []UserRoles{
			{
				UserID: event.User.ID,
				Roles:  []string{},
			},
		},
	}

	err = postRolesToHavenAPI(request)
	if err != nil {
		log.Printf("Error posting role update to Haven API: %v", err)
		return
	}
}

func postRolesToHavenAPI(request RoleUpdateRequest) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}

	url := config.HavenAPIBaseURL + "/guild-roles"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.HavenAPIToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to post roles to Haven API: %s", resp.Status)
	}

	return nil
}

type RoleUpdateRequest struct {
	GuildID string      `json:"guild_id"`
	Users   []UserRoles `json:"users"`
}

type UserRoles struct {
	UserID string   `json:"user_id"`
	Roles  []string `json:"roles"`
}
