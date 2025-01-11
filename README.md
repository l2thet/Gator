# Gator

## My Go and PostgreSQL Project for Tracking RSS Feeds

This project is a guided project from Boot.dev that I created and made adjustments to.

## Getting Started

These instructions will help you run this project on your local machine

### Prerequisites

- go 1.22.5 or later
- PostgreSQL 14.15 or later
- Goose (go install github.com/pressly/goose/v3/cmd/goose@latest)
- A custom json config file in your home directory named .gatorconfig.json

### Installation

1. **Clone the repository:**

```sh
git clone https://github.com/l2thet/gator.git
cd gator
cd sql/schema
goose postgres postgres://username:password@localhost:5432/gator up
cd ../..
go mod download
go install gator
```

### .gatorconfig.json
The only required attribute is db_url, it should point to your local postgres setup

```json
{
    "db_url":"postgres://username:password@localhost:5432/gator?sslmode=disable"
}
```

### Running the app
This is a CLI application, with commands that can take multiple arguements

Examples:

```sh
gator reset #This will reset the DB to a clean slate, be careful
gator register [username] #username being who will you will be adding to an RSS feed to track
gator login [username] #Set the current user to an existing user in the DB
gator addfeed [name of feed] [Url]
gator follow [url] #If a feed with a specific URL has already been added with addfeed even by another user this add the feed to the current user
gator following #This will list the current users RSS feeds
gator agg single #This will download all the current RSS feeds for the current user
gator browse [# of articles to display] #This will take an optional arguement, if not provided it will default to 2
```