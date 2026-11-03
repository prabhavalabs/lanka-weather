/**
 * The installed app: updates, the offline notice and the words the service
 * worker shows on a notification it could not read. The service worker
 * imports this file directly (it cannot load React), so keep it plain data.
 */
export const pwa = {
  update: {
    title: "A new version is ready",
    description: "Reload to use it. Nothing you have open will be lost.",
    reload: "Reload",
    later: "Later",
  },
  offline: {
    title: "You are offline",
    body: "The map shows the last saved data. Each layer shows how old it is.",
    back: "Back online. Updating the map.",
  },
  ready: "Lanka Weather now works offline.",
  push: {
    genericTitle: "Lanka Weather",
    genericBody: "There is a new update. Open the map to see it.",
  },
}
