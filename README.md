# todoist_task_sync

A simple sync script to pull queued work from around the web into my todo list.

## Features

* Syncs Freshdesk tickets into a specified Todoist list (and marks the task complete when the ticket is resolved, pending, or reassigned)

## Future features

* Sync Github pull requests for specific repos into a list
* Generalized sync framework for tasks

## Building

```make bin```

It'll create binaries for Linux, Mac, and Windows. I don't use anything but Linux right now, so I can't guarantee that it'll work anywhere but my specific Linux install.

## Usage

`./bin/linux-amd64/ttsync --help`

For me, the command looks something like this:

`./bin/linux-amd64/ttsync -fd-apikey="youwish" -fd-domain="mycompany" -todoist-apikey="lolno" -todoist-freshdesk-list="Customer support"`

