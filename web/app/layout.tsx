import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import { Footer } from "@/components/footer";
import { Navbar } from "@/components/navbar";
import { RouteMotion } from "@/components/route-motion";
import { ThemeProvider } from "@/components/theme-provider";
import { site } from "@/config/site";
import "./globals.css";

const inter = Inter({ variable: "--font-inter", subsets: ["latin"], display: "swap" });
const jetBrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: site.name, template: `%s · ${site.name}` },
  description: site.description,
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${jetBrainsMono.variable}`}
      data-navbar={site.navbar}
      suppressHydrationWarning
    >
      <body className="flex min-h-dvh flex-col">
        <ThemeProvider>
          <RouteMotion />
          <Navbar />
          {children}
          <Footer />
        </ThemeProvider>
      </body>
    </html>
  );
}
