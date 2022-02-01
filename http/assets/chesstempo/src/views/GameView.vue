<script setup lang="ts">
import { Chessground } from "chessground";
import { Config } from "chessground/config";
import { Api } from "chessground/api";
import { Key } from "chessground/types";
import { computed, reactive, onMounted, ref } from "vue";
import { useRouter, useRoute } from "vue-router";
import {
  GameIdentifier,
  fetchGame,
  GameState,
  moveGame,
  Turn,
  resignGame,
} from "@/client";

const router = useRouter();
const route = useRoute();
const id: GameIdentifier = Array.isArray(route.params.id)
  ? route.params.id[0]
  : route.params.id;

const board = ref<HTMLDivElement | null>(null);
const state: GameState = reactive({});

let ground: Api;

onMounted(async () => {
  await fetchGame(id).then((st) => {
    Object.assign(state, st);
    populate(id, st);
  });
});

const populate = (id: GameIdentifier, state: GameState) => {
  const dests = new Map();
  state.ValidMoves?.forEach((move) => {
    const from = move.slice(0, 2);
    const to = move.slice(2, 4);
    const item = dests.get(from);
    if (item) {
      item.push(to);
    } else {
      dests.set(from, [to]);
    }
  });

  const config: Config = {
    fen: state.FEN,
    draggable: {
      showGhost: true,
    },
    movable: {
      free: false,
      showDests: true,
      dests: dests,
    },
  };

  if (state.Turn === "User") {
    const color = state.Color === "White" ? "white" : "black";
    config.turnColor = color;
    if (config.movable) {
      config.movable.color = color;
    }
  }

  if (!board?.value) return;
  ground = Chessground(board?.value, config);
  ground.set({ movable: { events: { after: onMove(id) } } });
};

const onMove = (id: GameIdentifier) => {
  return (orig: Key, dest: Key) => {
    moveGame(id, orig, dest)
      .then(() => fetchGame(id))
      .then((state) => populate(id, state))
      .then(() => wait(500))
      .then(() => poll(id));
  };
};

const poll = async (id: GameIdentifier) => {
  let s = await fetchGame(id);
  Object.assign(state, s);
  populate(id, state);

  while (state.Turn === Turn.Machine) {
    await wait();
    s = await fetchGame(id);
    Object.assign(state, s);
    populate(id, state);
  }

  return state;
};

const wait = async (ms = 1000) => {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
};

const resign = async (id: GameIdentifier) => {
  await resignGame(id).then(() => {
    router.push({ name: "lobby" });
  });
};

const loaded = computed(() => {
  return Object.entries(state).length > 0;
});

const usersTurn = computed(() => {
  return state.Turn === Turn.User;
});

const done = computed(() => {
  return state.Outcome !== "*";
});
</script>

<template>
  <main class="game">
    <div class="panel" v-show="loaded">
      <h2 v-if="!done">
        You are playing as <i>{{ state.Color }}</i
        >.
        <br />
        <template v-if="usersTurn"> It's your turn! </template>
        <template v-else> Waiting... </template>
      </h2>

      <div class="done" v-if="done">Game is over! {{ state.Outcome }}</div>

      <div class="blue merida">
        <div ref="board" class="cg-board-wrap"></div>
      </div>

      <div class="actions" v-if="!done">
        <button @click="resign(id)">Resign</button>
      </div>
    </div>
  </main>
</template>

<style>
@import "@/assets/board.css";
</style>

<style scoped>
.user-color {
  background-color: #aaa;
  background-position: center / 80%;
  width: 400px;
  height: 400px;
}
.user-color-White {
  color: white;
}
.user-color-Black {
  color: black;
}

i {
  text-decoration: underline;
}

.done {
  text-align: center;
  background-color: yellowgreen;
  margin-bottom: 20px;
  padding: 20px;
  font-size: 20px;
}
</style>
