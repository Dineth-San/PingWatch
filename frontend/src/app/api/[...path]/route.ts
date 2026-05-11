import { NextRequest, NextResponse } from "next/server";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

async function proxy(req: NextRequest): Promise<NextResponse> {
  const target = `${API_URL}${req.nextUrl.pathname}${req.nextUrl.search}`;

  const res = await fetch(target, {
    method: req.method,
    headers: req.headers,
    body: req.method !== "GET" && req.method !== "HEAD" ? req.body : undefined,
    // @ts-expect-error duplex is required for streaming request bodies
    duplex: "half",
  });

  return new NextResponse(res.body, {
    status: res.status,
    statusText: res.statusText,
    headers: res.headers,
  });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const DELETE = proxy;
export const PATCH = proxy;
