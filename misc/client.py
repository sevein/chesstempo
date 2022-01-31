#!/usr/bin/env python3
"""Dumb command-line tool to mange chesstempo from the terminal.
"""

import os
import sys
import time
import random
import requests


init_url = f"http://127.0.0.1:9999/api/games"
info_url = f"{init_url}/%s"
move_url = f"{init_url}/%s/move/%s"
resign_url = f"{init_url}/%s/resign"


def info(identifier):
    resp = requests.get(info_url % (identifier,))
    try:
        resp.raise_for_status()
    except Exception as err:
        print(err)
    return resp.json()


def move(identifier, move):
    resp = requests.post(move_url % (identifier, move))
    resp.raise_for_status()
    return resp.json()


def start(fen, color):
    payload = {}
    if fen:
        payload["fen"] = fen
    if color:
        payload["color"] = color
    resp = requests.post(init_url, json=payload)
    resp.raise_for_status()
    return resp.json()


def list_all():
    resp = requests.get(init_url)
    resp.raise_for_status()
    return resp.json()


def resign(identifier):
    resp = requests.post(resign_url % (identifier,))
    resp.raise_for_status()
    return resp.json()


def resign_all():
    for identifier in list_all():
        resign(identifier)


def draw(info, *, clear=True):
    if clear:
        os.system("cls" if os.name == "nt" else "clear")
    print(info["Board"])


fen = None
if "--demo" in sys.argv:
    # Final decisive game of the 2014 Carlsen vs. Anand World Championship match.
    fen = "8/4b3/4P3/1k4P1/8/ppK5/8/4R3 b - - 1 45"

if "--resign" in sys.argv:
    if len(sys.argv) < 2:
        sys.exit("Please include the identifier of the game.")
    resign(sys.argv[2])
    sys.exit(0)

if "--resign-all" in sys.argv:
    resign_all()
    sys.exit(0)

if "--list" in sys.argv:
    for item in list_all():
        print(item)
    sys.exit(0)

identifier = None
if "--continue" in sys.argv:
    identifier = sys.argv[2]

# Start new game.
if not identifier:
    color = None
    if "--white" in sys.argv:
        color = "w"
    elif "--black" in sys.argv:
        color = "b"
    resp = start(fen, color)
    identifier = resp["id"]

print("Game started", identifier)

while True:
    try:
        inf = info(identifier)
    except Exception as err:
        print("Error!", err)
        break
    else:
        draw(inf)
        print(f"Game: {identifier}")

    # Game is over.
    if inf["Outcome"] != "*":
        print("Done!")
        print(f"Outcome: {inf['Outcome']}")
        break

    # Move
    m = random.choice(inf["ValidMoves"])
    ret = move(identifier, m)
    print(f"Last move: {m}")
    print(f"Outcome: {inf['Outcome']}")
    print(inf["Turn"], inf["Color"])

    time.sleep(1)
