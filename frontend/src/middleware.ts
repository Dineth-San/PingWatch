import { NextRequest, NextResponse } from "next/server";

const PROTECTED = ["/dashboard", "/monitors"];
const AUTH_ONLY = ["/login", "/register"];

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  const hasToken = req.cookies.has("token");

  const isProtected = PROTECTED.some(p => pathname === p || pathname.startsWith(p + "/"));
  const isAuthOnly = AUTH_ONLY.some(p => pathname === p || pathname.startsWith(p + "/"));

  if (isProtected && !hasToken) {
    return NextResponse.redirect(new URL("/login", req.url));
  }

  if (isAuthOnly && hasToken) {
    return NextResponse.redirect(new URL("/dashboard", req.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*", "/monitors/:path*", "/login", "/register"],
};
