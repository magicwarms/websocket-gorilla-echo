import { constraints } from "@/config"

const displayMediaOptions = {
    video: {
      displaySurface: "browser",
    },
    audio: {
      suppressLocalAudioPlayback: false,
    },
    preferCurrentTab: false,
    selfBrowserSurface: "exclude",
    systemAudio: "include",
    surfaceSwitching: "include",
    monitorTypeSurfaces: "include",
  };
export const openMediaDevices = async () => {
    return await navigator.mediaDevices.getDisplayMedia(displayMediaOptions as DisplayMediaStreamOptions)
}
