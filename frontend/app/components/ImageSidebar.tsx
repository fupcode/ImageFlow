"use client";

import { useState } from "react";
import type { MouseEvent } from "react";
import Image from "next/image";
import { motion, AnimatePresence } from "framer-motion";
import ImageModal from "../components/ImageModal";
import ImageLightbox from "./ImageLightbox";
import { getFullUrl } from "../utils/baseUrl";
import ContextMenu, { ContextMenuGroup } from "./ContextMenu";
import { showToast } from "./ToastContainer";
import type { ImageFile, UploadResult } from "../types";
import {
  copyOriginalUrl,
  copyWebpUrl,
  copyAvifUrl,
  copyMarkdownLink,
  copyHtmlImgTag,
  copyToClipboard,
} from "../utils/copyImageUtils";
import {
  ImageIcon,
  Cross1Icon,
  ExclamationTriangleIcon,
  ClipboardCopyIcon,
  FileIcon,
  CopyIcon,
  CheckIcon,
  EyeOpenIcon,
} from "./ui/icons";

interface ImageSidebarProps {
  isOpen: boolean;
  results: UploadResult[];
  onClose: () => void;
  onDelete?: (id: string) => Promise<void>;
  onTagsUpdate?: (id: string, tags: string[]) => void;
}

type UploadImage = UploadResult & { status: "success" };

function isSuccessfulUpload(image: UploadResult): image is UploadImage {
  return image.status === "success";
}

export default function ImageSidebar({
  isOpen,
  results,
  onClose,
  onDelete,
  onTagsUpdate,
}: ImageSidebarProps) {
  const [selectedImage, setSelectedImage] = useState<UploadResult | null>(null);
  const [previewImage, setPreviewImage] = useState<UploadImage | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [tab, setTab] = useState<"all" | "success" | "error">("all");
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [contextMenu, setContextMenu] = useState<{
    isOpen: boolean;
    x: number;
    y: number;
    image: UploadImage | null;
  }>({
    isOpen: false,
    x: 0,
    y: 0,
    image: null,
  });

  const successResults = results.filter(
    (result) => result.status === "success"
  );
  const errorResults = results.filter((result) => result.status === "error");

  // 根据当前标签确定要显示的结果
  const displayResults =
    tab === "all" ? results : tab === "success" ? successResults : errorResults;

  const handleImageClick = (image: UploadResult) => {
    if (isSuccessfulUpload(image)) {
      setPreviewImage(image);
    }
  };

  const handleShowDetails = (image: UploadImage) => {
    setSelectedImage(image);
    setShowModal(true);
  };

  const handleCloseModal = () => {
    setShowModal(false);
  };

  const handleTagsUpdate = (id: string, tags: string[]) => {
    setSelectedImage((prevImage) =>
      prevImage && prevImage.id === id ? { ...prevImage, tags } : prevImage
    );
    onTagsUpdate?.(id, tags);
  };

  const toImageFile = (image: UploadImage): ImageFile => ({
    id: image.id || image.filename,
    filename: image.filename,
    url: image.urls?.webp || image.urls?.original || "",
    format: image.format || "image",
    orientation: image.orientation || "landscape",
    size: 0,
    path: image.path || "",
    storageType: "local",
    tags: image.tags,
    urls: image.urls,
  });

  const handleQuickCopy = async (image: UploadResult, event: MouseEvent) => {
    event.stopPropagation();

    if (!isSuccessfulUpload(image)) return;

    const url = image.urls?.webp;
    if (!url) {
      showToast("没有可复制的WebP链接", "error");
      return;
    }

    const success = await copyToClipboard(getFullUrl(url));
    if (success) {
      const id = image.id || image.filename;
      setCopiedId(id);
      showToast("复制成功", "success");
      window.setTimeout(() => setCopiedId(null), 1200);
    } else {
      showToast("复制失败", "error");
    }
  };

  const handleCopy = async (image: UploadImage, type: string) => {
    const imageFile = toImageFile(image);
    let success = false;

    try {
      switch (type) {
        case "original":
          success = await copyOriginalUrl(imageFile);
          break;
        case "webp":
          success = await copyWebpUrl(imageFile);
          break;
        case "avif":
          success = await copyAvifUrl(imageFile);
          break;
        case "markdown":
          success = await copyMarkdownLink(imageFile);
          break;
        case "html":
          success = await copyHtmlImgTag(imageFile);
          break;
      }

      showToast(success ? "复制成功" : "复制失败", success ? "success" : "error");
    } catch (error) {
      showToast("复制失败", "error");
      console.error("复制错误:", error);
    }
  };

  const handleContextMenu = (image: UploadResult, event: MouseEvent) => {
    if (!isSuccessfulUpload(image)) return;

    event.preventDefault();
    setContextMenu({
      isOpen: true,
      x: event.clientX,
      y: event.clientY,
      image,
    });
  };

  const closeContextMenu = () => {
    setContextMenu((prev) => ({ ...prev, isOpen: false }));
  };

  const menuImage = contextMenu.image;
  const menuGroups: ContextMenuGroup[] = menuImage
    ? [
        {
          id: "details",
          items: [
            {
              id: "details",
              label: "查看详细信息",
              onClick: () => handleShowDetails(menuImage),
              icon: <EyeOpenIcon className="h-4 w-4" />,
            },
          ],
        },
        {
          id: "copy",
          items: [
            {
              id: "copy-original",
              label: `复制原始链接 (${(menuImage.format || "original").toUpperCase()})`,
              onClick: () => handleCopy(menuImage, "original"),
              icon: <ClipboardCopyIcon className="h-4 w-4" />,
              disabled: !menuImage.urls?.original,
            },
            {
              id: "copy-webp",
              label: "复制WebP链接",
              onClick: () => handleCopy(menuImage, "webp"),
              icon: <ClipboardCopyIcon className="h-4 w-4" />,
              disabled: !menuImage.urls?.webp,
            },
            ...(menuImage.urls?.avif
              ? [
                  {
                    id: "copy-avif",
                    label: "复制AVIF链接",
                    onClick: () => handleCopy(menuImage, "avif"),
                    icon: <ClipboardCopyIcon className="h-4 w-4" />,
                  },
                ]
              : []),
          ],
        },
        {
          id: "format",
          items: [
            {
              id: "copy-markdown",
              label: "复制Markdown标签",
              onClick: () => handleCopy(menuImage, "markdown"),
              icon: <FileIcon className="h-4 w-4" />,
            },
            {
              id: "copy-html",
              label: "复制HTML标签",
              onClick: () => handleCopy(menuImage, "html"),
              icon: <FileIcon className="h-4 w-4" />,
            },
          ],
        },
      ]
    : [];

  return (
    <>
      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ x: "100%" }}
            animate={{ x: 0 }}
            exit={{ x: "100%" }}
            transition={{ type: "spring", damping: 30, stiffness: 300 }}
            className="fixed top-0 right-0 w-full sm:w-96 h-full bg-white dark:bg-slate-900 shadow-xl z-30 border-l border-slate-200 dark:border-slate-700 overflow-hidden flex flex-col"
          >
            {/* 侧边栏头部 */}
            <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-slate-700 bg-gradient-to-r from-indigo-500 to-purple-600 text-white">
              <h2 className="text-lg font-semibold flex items-center">
                <ImageIcon className="h-5 w-5 mr-2 text-white opacity-90" />
                上传结果 ({results.length})
              </h2>
              <button
                onClick={onClose}
                className="p-2 rounded-full hover:bg-white/10 transition-colors"
              >
                <Cross1Icon className="h-5 w-5" />
              </button>
            </div>

            {/* 标签切换 */}
            <div className="flex border-b border-slate-200 dark:border-slate-700">
              <button
                onClick={() => setTab("all")}
                className={`flex-1 py-3 px-4 text-sm font-medium transition-colors relative ${
                  tab === "all"
                    ? "text-indigo-600 dark:text-indigo-400"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200"
                }`}
              >
                全部 ({results.length})
                {tab === "all" && (
                  <motion.div
                    layoutId="tab-indicator"
                    className="absolute bottom-0 left-0 right-0 h-0.5 bg-indigo-600 dark:bg-indigo-400"
                  />
                )}
              </button>
              <button
                onClick={() => setTab("success")}
                className={`flex-1 py-3 px-4 text-sm font-medium transition-colors relative ${
                  tab === "success"
                    ? "text-green-600 dark:text-green-400"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200"
                }`}
              >
                成功 ({successResults.length})
                {tab === "success" && (
                  <motion.div
                    layoutId="tab-indicator"
                    className="absolute bottom-0 left-0 right-0 h-0.5 bg-green-600 dark:bg-green-400"
                  />
                )}
              </button>
              <button
                onClick={() => setTab("error")}
                className={`flex-1 py-3 px-4 text-sm font-medium transition-colors relative ${
                  tab === "error"
                    ? "text-red-600 dark:text-red-400"
                    : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200"
                }`}
              >
                失败 ({errorResults.length})
                {tab === "error" && (
                  <motion.div
                    layoutId="tab-indicator"
                    className="absolute bottom-0 left-0 right-0 h-0.5 bg-red-600 dark:bg-red-400"
                  />
                )}
              </button>
            </div>

            {/* 侧边栏内容 */}
            <div className="flex-1 overflow-y-auto p-4">
              {displayResults.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-full text-center text-slate-500 dark:text-slate-400 p-6">
                  <ImageIcon className="h-16 w-16 mb-4 text-slate-300 dark:text-slate-600" />
                  <p className="text-lg font-medium mb-2">暂无图片</p>
                  <p className="text-sm">
                    {tab === "all"
                      ? "上传完成的图片将会显示在这里"
                      : tab === "success"
                      ? "没有成功上传的图片"
                      : "没有上传失败的图片"}
                  </p>
                </div>
              ) : (
                <div className="space-y-4">
                  {/* 显示选定的结果 */}
                  <div className="grid grid-cols-2 gap-3">
                    {displayResults.map((result, index) => (
                      <motion.div
                        key={result.id || `${tab}-${index}`}
                        initial={{ opacity: 0, scale: 0.9 }}
                        animate={{ opacity: 1, scale: 1 }}
                        transition={{ delay: index * 0.05 }}
                        className={`relative rounded-lg overflow-hidden border ${
                          result.status === "success"
                            ? "border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800"
                            : "border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20"
                        } shadow-sm hover:shadow-md transition-all cursor-pointer group`}
                        onClick={() =>
                          result.status === "success" &&
                          handleImageClick(result)
                        }
                        onContextMenu={(event) => handleContextMenu(result, event)}
                      >
                        {result.status === "success" ? (
                          <>
                            <div className="aspect-square relative bg-slate-50 dark:bg-slate-900">
                              {result.urls?.original && (
                                <Image
                                  src={getFullUrl(result.urls.thumb || result.urls.webp)}
                                  alt={result.filename}
                                  fill
                                  className="object-cover group-hover:scale-105 transition-transform duration-300"
                                  sizes="(max-width: 768px) 50vw, 33vw"
                                />
                              )}
                              <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
                              <div className="absolute top-1 left-1">
                                <span className="text-xs px-1.5 py-0.5 bg-green-500/80 text-white rounded-full">
                                  完成
                                </span>
                              </div>
                              <button
                                type="button"
                                onClick={(event) => handleQuickCopy(result, event)}
                                className="absolute top-1 right-1 p-1.5 rounded-full bg-white/20 text-white backdrop-blur-sm opacity-0 group-hover:opacity-100 hover:bg-white/40 transition-all"
                                title="复制WebP链接"
                              >
                                {copiedId === (result.id || result.filename) ? (
                                  <CheckIcon className="h-4 w-4 text-green-300" />
                                ) : (
                                  <CopyIcon className="h-4 w-4" />
                                )}
                              </button>
                              <div className="absolute bottom-0 left-0 right-0 p-2 text-white transform translate-y-full group-hover:translate-y-0 transition-transform duration-300">
                                <p
                                  className="text-xs truncate"
                                  title={result.filename}
                                >
                                  {result.filename}
                                </p>
                                {result.expiryTime && (
                                  <p className="text-xs mt-1">
                                    <span className="bg-yellow-500/80 text-white px-1 py-0.5 rounded text-[10px]">
                                      过期时间:{" "}
                                      {new Date(
                                        result.expiryTime
                                      ).toLocaleString()}
                                    </span>
                                  </p>
                                )}
                              </div>
                            </div>
                          </>
                        ) : (
                          <div className="relative p-3 pt-8 h-full flex flex-col">
                            <span className="absolute top-1 left-1 text-xs px-1.5 py-0.5 bg-red-500/85 text-white rounded-full">
                              失败
                            </span>
                            <div className="flex items-start space-x-2">
                              <ExclamationTriangleIcon className="h-5 w-5 text-red-500 flex-shrink-0 mt-0.5" />
                              <div>
                                <p className="font-medium text-sm text-red-600 dark:text-red-400">
                                  {result.filename}
                                </p>
                                <p className="text-xs text-red-500 dark:text-red-300 mt-1">
                                  {result.message}
                                </p>
                              </div>
                            </div>
                          </div>
                        )}
                      </motion.div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>

      {/* 图片详情模态框 */}
      <ImageModal
        image={selectedImage && selectedImage.status === "success" ? selectedImage as any : null}
        isOpen={showModal}
        onClose={handleCloseModal}
        onDelete={onDelete}
        onTagsUpdate={handleTagsUpdate}
      />

      <ContextMenu
        items={menuGroups}
        isOpen={contextMenu.isOpen}
        x={contextMenu.x}
        y={contextMenu.y}
        onClose={closeContextMenu}
      />

      <ImageLightbox
        src={
          previewImage
            ? getFullUrl(previewImage.urls?.thumb || previewImage.urls?.webp || previewImage.urls?.original || "")
            : ""
        }
        alt={previewImage?.filename || ""}
        isOpen={Boolean(previewImage)}
        onClose={() => setPreviewImage(null)}
      />

      {/* 背景遮罩 */}
      <AnimatePresence>
        {isOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/20 dark:bg-black/50 backdrop-blur-sm z-20 sm:block hidden"
            onClick={onClose}
          />
        )}
      </AnimatePresence>
    </>
  );
}
