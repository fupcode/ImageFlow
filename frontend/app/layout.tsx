import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "ImageFlow - 图片管理",
  description: "一个简单而强大的图片管理工具",
  icons: {
    icon: [
      { url: "/static/favicon.ico", sizes: "any" },
      { url: "/static/favicon.svg", type: "image/svg+xml" },
      { url: "/static/favicon-48.png", sizes: "48x48", type: "image/png" },
      { url: "/static/favicon-32.png", sizes: "32x32", type: "image/png" },
      { url: "/static/favicon-16.png", sizes: "16x16", type: "image/png" },
    ],
    apple: [
      { url: "/static/favicon-48.png", sizes: "48x48", type: "image/png" },
    ],
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh-CN" className="light">
      <body className="page-bg py-10 transition-colors duration-300 light-mode font-bold">
        {/* 动态背景 */}
        <div className="animated-bg">
          <div className="bubble"></div>
          <div className="bubble"></div>
          <div className="bubble"></div>
          <div className="bubble"></div>
          <div className="bubble"></div>
        </div>

        {children}

      </body>
    </html>
  );
}
