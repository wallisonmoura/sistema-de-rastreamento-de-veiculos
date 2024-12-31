import { NextRequest, NextResponse } from "next/server";
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ routeId: string }> }
) {
  const { routeId } = await params;
  console.log('pegou ',routeId);
  const response = await fetch(`${process.env.NEST_API_URL}/routes/${routeId}`, {
    cache: "force-cache",
    next: {
      tags: [`routes-${routeId}`, "routes"],
    },
  });
  const data = await response.json();
  return NextResponse.json(data);
}