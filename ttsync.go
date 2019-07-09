package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	freshdesk "github.com/cweagans/go-freshdesk"
	"github.com/cweagans/go-freshdesk/querybuilder"
	"github.com/kobtea/go-todoist/todoist"
)

var (
	freshdeskDomain       string
	freshdeskApikey       string
	todoistAPIKey         string
	todoistFreshdeskList  string
	freshdeskCustomDomain string
)

func init() {
	flag.StringVar(&freshdeskDomain, "fd-domain", os.Getenv("FRESHDESK_DOMAIN"), "____.freshdesk.com -- the domain for your support portal")
	flag.StringVar(&freshdeskApikey, "fd-apikey", os.Getenv("FRESHDESK_APIKEY"), "The API key provided on your Freshdesk 'Profile Settings' page")
	flag.StringVar(&todoistAPIKey, "todoist-apikey", os.Getenv("TODOIST_APIKEY"), "Your Todoist API key")
	flag.StringVar(&todoistFreshdeskList, "todoist-freshdesk-list", os.Getenv("TODOIST_FRESHDESK_LIST"), "The list to which Freshdesk tickets should be added")
	flag.StringVar(&freshdeskCustomDomain, "fd-custom-domain", os.Getenv("FRESHDESK_CUSTOM_DOMAIN"), "A custom domain to use for ticket links in tasks. You need to set this if your Freshdesk is using a custom domain. ")
	flag.Parse()

	if freshdeskCustomDomain == "" {
		freshdeskCustomDomain = freshdeskDomain + ".freshdesk.com"
	}
}

func main() {
	// Create a new logger for freshdesk ops.
	fdlogger := log.New(os.Stdout, "[freshdesk] ", 0)
	fdlogger.Println("Downloading tickets")
	client := freshdesk.Init(freshdeskDomain, freshdeskApikey, &freshdesk.ClientOptions{Logger: fdlogger})

	// Get info about the current agent so that we can get a list of their tickets.
	currentAgent, err := client.Agents.Me()
	if err != nil {
		fdlogger.Panic(err)
	}

	fdlogger.Printf("Current agent: %s (%d)", currentAgent.Contact.Name, currentAgent.ID)
	fdlogger.Println("Finding tickets for current agent")

	// Build the ticket query.
	agentquery := querybuilder.Parameter("agent_id").Equals(currentAgent.ID)
	statusquery := querybuilder.Parameter("status").Equals(int(freshdesk.StatusOpen))
	combinedquery := querybuilder.AllOf(agentquery, statusquery)

	// Download ticket data.
	tickets, err := client.Tickets.Search(combinedquery)
	if err != nil {
		fdlogger.Panic(err)
	}

	// Create a new logger for todoist ops.
	tdlogger := log.New(os.Stdout, "[todoist] ", 0)
	tdlogger.Println("Downloading todoist account data")

	// Create a new client.
	t, _ := todoist.NewClient("", todoistAPIKey, "", "", tdlogger)
	ctx := context.Background()
	err = t.FullSync(ctx, []todoist.Command{})
	if err != nil {
		tdlogger.Panic(err)
	}

	// Find the freshdesk list
	project := t.Project.FindOneByName(todoistFreshdeskList)
	if project == nil {
		tdlogger.Panicf("Couldn't find the Freshdesk list '%s' in Todoist", todoistFreshdeskList)
	}
	tdlogger.Printf("Found target project %s (id: %s)", project.Name, project.ID)

	// Create tasks for tickets that are open and assigned to the current user.
	for _, ticket := range tickets.Results {
		tasktext := fmt.Sprintf("#%d: [%s](https://%s/a/tickets/%d)", ticket.ID, ticket.Subject, freshdeskCustomDomain, ticket.ID)

		tditems := t.Item.FindByContent(tasktext)
		if len(tditems) == 0 {
			tdlogger.Printf("Task not found for ticket %d. Creating...\n", ticket.ID)
			tditem := todoist.Item{
				Content:   tasktext,
				ProjectID: project.ID,
			}
			_, err = t.Item.Add(tditem)
			if err != nil {
				tdlogger.Fatalln("Could not create task!")
			}

			t.Commit(ctx)
		}
	}

	// Get the list of project items.
	todoistitems := t.Item.FindByProjectIDs([]todoist.ID{project.ID})

	// Now, loop through the Todoist list to find tickets that are closed or pending
	// and check them off so that Todoist doesn't bug me about them.
	var ticketID = regexp.MustCompile(`^#[0-9]+`)
	for _, item := range todoistitems {
		// Skip items that don't match.
		if !ticketID.MatchString(item.Content) {
			continue
		}

		ticketIDString := ticketID.FindString(item.Content)
		ticketNumber := ticketIDString[1:]
		ticketInt, err := strconv.Atoi(ticketNumber)
		if err != nil {
			tdlogger.Fatalln("Could not convert ticket number to int")
		}

		fdlogger.Printf("Looking up ticket #%d", ticketInt)

		ticket, err := client.Tickets.View(ticketInt)
		if err != nil {
			fdlogger.Println("Could not find ticket. Marking item closed (ticket may have been deleted)")
		}

		// If the ticket is closed or has been reassigned, mark the task complete.
		if err != nil || ticket.Status == int(freshdesk.StatusPending) || ticket.Status == int(freshdesk.StatusResolved) || ticket.Status == int(freshdesk.StatusClosed) || ticket.ResponderID != currentAgent.ID {
			tdlogger.Printf("Marking task for resolved ticket #%d complete", ticket.ID)
			err := t.Item.Close(item.ID)
			if err != nil {
				tdlogger.Panicln("Could not mark task completed!")
			}
		}
	}

	// Do a final sync to get everything consistent.
	tdlogger.Println("Syncing updated data to Todoist")
	err = t.Commit(ctx)
	if err != nil {
		tdlogger.Panic(err)
	}

	tdlogger.Println("Done")
}
