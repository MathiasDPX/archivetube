import duckdb
import requests
import numpy as np

API_KEY = "redacted"
URL = "https://ai.hackclub.com/proxy/v1/embeddings"
MODEL = "qwen/qwen3-embedding-8b"

duckdb.sql('INSTALL sqlite;')
conn = duckdb.connect('database.db')

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
    return data

def cosine_similarity(a, b):
    a = np.array(a)
    b = np.array(b)
    return np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b))

class Video:
    def __init__(self, title, description):
        self.title = title
        self.desscription = description
    
        self.title_vec = get_embedding(title)
        self.description_vec = get_embedding(self.desscription)


videos:list[Video] = []
for video in conn.sql("SELECT title, description FROM videos LIMIT 10;").fetchall():
    videos.append(Video(video[0], video[1]))
    print(video[0])

def search(query, top_k=3):
    query_embedding = get_embedding(query)

    scores = []
    for i, video in enumerate(videos):
        desc_score = cosine_similarity(query_embedding, video.description_vec)
        title_score = cosine_similarity(query_embedding, video.title_vec)
        score = (title_score * 1) + (desc_score * 0.5) / 1.5
        scores.append((videos[i], score))

    results = sorted(scores, key=lambda x: x[1], reverse=True)
    return results[:top_k]

print("---")
query = input("Query> ")

for result in search(query):
    video, score = result
    print(f"{score:.4f}: {video.title}")