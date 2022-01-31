<script setup lang="ts">
import { onMounted, reactive } from "vue";
import { useRouter } from "vue-router";
import { startGame, listGames, Color, GameIdentifiers } from "@/client";

const router = useRouter();

const games = reactive({
  items: [] as GameIdentifiers,
});

const start = (color?: Color) => {
  startGame(color).then((id) => {
    router.push({ name: "game", params: { id: id } });
  });
};

onMounted(() => {
  listGames().then((identifiers) => (games.items = identifiers));
});
</script>

<template>
  <h2 class="text-3xl font-bold underline">Lobby</h2>
  <div class="games">
    <div class="game" v-for="id in games.items" :key="id">
      <RouterLink :to="{ name: 'game', params: { id: id } }"
        >&raquo; {{ id }}</RouterLink
      >
    </div>
  </div>
  <div class="actions">
    <div class="heading">Start game as...</div>
    <button class="btn btn-blue" @click="start(Color.White)">White</button>
    <button class="btn btn-blue" @click="start(Color.Black)">Black</button>
    <button class="btn btn-blue" @click="start()">Random</button>
  </div>
</template>

<style scoped>
.games {
  margin: 0 auto;
}

.game {
  border-radius: 4px;
  text-align: center;
  margin-bottom: 10px;
  font-family: monospace;
}

.game a {
  display: block;
  color: white;
  text-decoration: none;
  background-color: #666;
  padding: 10px;
  font-size: 1.5em;
  text-align: left;
}

.game a:hover {
  background-color: #fff;
  color: #333;
}
</style>
