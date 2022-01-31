export type GameIdentifier = string;

export type GameIdentifiers = Array<string>;

export enum Color {
  White = "w",
  Black = "b",
}

export enum Turn {
  User = "User",
  Machine = "Machine",
}

export interface GameState {
  FEN?: string;
  ValidMoves?: string[];
  Turn?: Turn;
  Color?: string;
  Outcome?: string;
}

interface StartGameRequest {
  color?: Color;
  fen?: string;
}

const listGames = async (): Promise<GameIdentifiers> => {
  const resp = await window.fetch("/api/games", { method: "GET" });
  const data = await resp.json();
  if (!resp.ok) {
    return Promise.reject(new Error(":-("));
  }
  return data;
};

const fetchGame = async (id: GameIdentifier): Promise<GameState> => {
  const resp = await window.fetch("/api/games/" + id, { method: "GET" });
  const state = await resp.json();
  if (!resp.ok) {
    return Promise.reject(new Error(":-("));
  }
  return state;
};

const resignGame = async (id: GameIdentifier) => {
  const resp = await window.fetch("/api/games/" + id + "/resign", {
    method: "POST",
  });
  const data = await resp.json();
  if (!resp.ok) {
    return Promise.reject(new Error(":-("));
  }
  return data;
};

const startGame = async (
  color?: Color,
  fen?: string
): Promise<GameIdentifier> => {
  const body: StartGameRequest = { color, fen };
  const resp = await window.fetch("/api/games", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const data = await resp.json();
  if (!resp.ok) {
    return Promise.reject(new Error(":-("));
  }
  return data.id;
};

const moveGame = async (
  id: GameIdentifier,
  orig: string,
  dest: string
): Promise<object> => {
  const url = "/api/games/" + id + "/move/" + orig + dest;
  const resp = await window.fetch(url, { method: "POST" });
  const data = await resp.json();
  if (!resp.ok) {
    return Promise.reject(new Error(":-("));
  }
  return data.id;
};

export { fetchGame, resignGame, listGames, startGame, moveGame };
