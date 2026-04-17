# Embedding migration

Small script to add embed all videos in the database

**Should I run this?**: Only if you already archived videos with smart-search off and switched smart-search on

### How to run

1. Stop your container
2. Backup the database
3. Copy the database file to `./embedding-migration/database.db`
4. Create a `apikey` with your HackClub AI inside
5. Install requirements with `pip install -r requirements.txt`
6. Run `python main.py`
7. Copy `database.db` back to the database

### dev

You can limit the number of videos to embed by adding `--limit ?` and if you want to query through the embedded videos, add `--test`