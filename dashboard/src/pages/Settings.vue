<template>
  <div>
    <h1 class="text-2xl font-semibold mb-4">Settings</h1>
    <div class="card">
      <form class="grid grid-cols-1 md:grid-cols-2 gap-4" @submit.prevent>
        <div>
          <label for="displayName" class="block text-sm font-medium mb-1">Display name</label>
          <input id="displayName" type="text" autocomplete="name" class="form-control" />
        </div>
        <div>
          <label for="company" class="block text-sm font-medium mb-1">Company</label>
          <input id="company" type="text" autocomplete="organization" class="form-control" />
        </div>
        <div>
          <label for="timezone" class="block text-sm font-medium mb-1">Timezone</label>
          <select id="timezone" class="form-control bg-white">
            <option value="">Select…</option>
            <option>UTC</option>
            <option>America/Los_Angeles</option>
            <option>Europe/Berlin</option>
          </select>
        </div>
        <div class="flex items-center gap-2">
          <input id="emails" type="checkbox" class="h-4 w-4" />
          <label for="emails" class="text-sm">Email notifications</label>
        </div>
        <div class="md:col-span-2">
          <button type="submit" class="w-full md:w-auto btn-primary">Save changes</button>
        </div>
      </form>
    </div>
  </div>
</template>
