import duckdb
import requests
import numpy as np

API_KEY = open('apikey', 'r').read().strip()
URL = "https://ai.hackclub.com/proxy/v1/embeddings"
MODEL = "qwen/qwen3-embedding-8b"

conn = duckdb.connect('database.db')
conn.execute("INSTALL vss; LOAD vss;")
conn.execute("SET hnsw_enable_experimental_persistence = true;")

conn.execute('''
    CREATE TABLE IF NOT EXISTS vec_titles (
        title TEXT PRIMARY KEY,
        title_vec FLOAT[384],
        description_vec FLOAT[384]
    )
''')

def get_embedding(text):
    response = requests.post(
        URL,
        headers={
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json",
        },
        json={
            "model": MODEL,
            "input": text,
            "dimensions": 384
        }
    )

    data = response.json()["data"][0]["embedding"]
    return np.array(data, dtype=np.float32)

def embed(limit=10**10):
    for video in conn.execute("SELECT title, description FROM videos LIMIT ?", (limit,)).fetchall():
        title, description = video
        exists = conn.execute("SELECT title FROM vec_titles WHERE title = ?", (title,)).fetchone()
        if not exists:
            title_vec = get_embedding(title)
            description_vec = get_embedding(description)
            conn.execute(
                "INSERT INTO vec_titles (title, title_vec, description_vec) VALUES (?, ?::FLOAT[384], ?::FLOAT[384])",
                (title, title_vec.tolist(), description_vec.tolist())
            )
        print(title)

def search(query, top_k=3):
    query_vec = get_embedding(query)

    # KNN search on title_vec
    title_results = conn.execute('''
        SELECT title, array_cosine_distance(title_vec::FLOAT[384], ?::FLOAT[384]) AS distance
        FROM vec_titles
        ORDER BY distance
        LIMIT ?
    ''', (query_vec.tolist(), top_k * 2)).fetchall()

    # KNN search on description_vec
    desc_results = conn.execute('''
        SELECT title, array_cosine_distance(description_vec::FLOAT[384], ?::FLOAT[384]) AS distance
        FROM vec_titles
        ORDER BY distance
        LIMIT ?
    ''', (query_vec.tolist(), top_k * 2)).fetchall()

    # Combine scores
    scores = {}
    for title, dist in title_results:
        scores[title] = dist * 1.0
    for title, dist in desc_results:
        scores[title] = scores.get(title, 0) + dist * 0.5

    results = sorted(scores.items(), key=lambda x: x[1])
    return results[:top_k]

if __name__ == "__main__":
    # homemade argparser
    import sys
    
    limit = 10**10
    if "--limit" in sys.argv:
        limit = int(sys.argv[sys.argv.index("--limit") + 1])
        
    embed(limit)
    
    if "--test" in sys.argv:
        if limit != 0:
            print("---")
        query = input("Query> ")

        for title, score in search(query):
            print(f"{score:.4f}: {title}")