<script setup lang="ts">
import { RouterLink, RouterView } from "vue-router";
</script>

<template>
  <div class="container">
    <header>
      <RouterLink :to="{ name: 'lobby' }">
        ♘
        <span>chesstempo</span>
      </RouterLink>
    </header>
    <RouterView />
  </div>
</template>

<style>
@import "@/assets/base.css";
</style>
