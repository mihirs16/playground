import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  pi.on("before_provider_request", (event) => {
    const payload = event.payload;
    if (payload && typeof payload === 'object' && 'model' in payload) {
      // Inject chat_template_kwargs to disable thinking
      payload.chat_template_kwargs = { "enable_thinking": false };
    }
  });
}
