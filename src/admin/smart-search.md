# Smart search

ArchiveTube search feature can be enhance by embedding video's title and description. This is done using `qwen/qwen3-embedding-8b` through OpenRouter. When using smart search, each search query gets embedded server-side ; this could be expensive if you have a public instance.

> [!WARNING]
> If you already have archived videos and smart-search was disabled, you need to follow the [Migration guide](#migration)


Currently, the only supported OpenRouter backend is [HackClub AI](https://ai.hackclub.com), but this will change in the future (probably less than 2 days after this commit).
You simply need to add this in your config and replace the apikey by your HackClub AI api key and restart your container

```toml
[smart_search]
enabled = true
apikey = "sk-hc-v1-"
```


## Migration

If you already have archived videos that were archived without smart-search enabled, you'll need to embed them. 

1. Clone ArchiveTube on your PC
2. Open the `embedding-migration` folder
3. Stop your container
4. Backup the database
5. Copy the database file to `./embedding-migration/database.db`
6. Create a `apikey` with your HackClub AI inside
7. Install requirements with `pip install -r requirements.txt`
8. Run `python main.py`
9. Copy `database.db` back to the database

This will embed ALL titles and descriptions, this can be expensive if you have a lot of videos archived.