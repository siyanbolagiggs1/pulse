import { ImageResponse } from "next/og";

export const runtime = "edge";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default async function Image() {
  return new ImageResponse(
    (
      <div
        style={{
          height: "100%",
          width: "100%",
          display: "flex",
          flexDirection: "column",
          alignItems: "flex-start",
          justifyContent: "center",
          background: "linear-gradient(135deg, #4338ca 0%, #6366f1 60%, #818cf8 100%)",
          padding: "80px",
        }}
      >
        <div
          style={{
            fontSize: 96,
            fontWeight: 700,
            color: "white",
            letterSpacing: -2,
          }}
        >
          Pulse
        </div>
        <div
          style={{
            fontSize: 36,
            color: "rgba(255,255,255,0.9)",
            marginTop: 16,
            maxWidth: 900,
          }}
        >
          Community-powered social promotion — businesses run repost
          campaigns, promoters earn money sharing them.
        </div>
      </div>
    ),
    { ...size }
  );
}
