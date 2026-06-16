"use client";

import { useState, useEffect } from "react";
import { AnimatePresence } from "framer-motion";
import { ImageFile } from "../types";
import { ImageData } from "../types/image";
import { api } from "../utils/request";
import { ImageInfo } from "./ImageInfo";
import { ImageUrls } from "./ImageUrls";
import { DeleteConfirm } from "./DeleteConfirm";
import {
  CheckIcon,
  Cross1Icon,
  TrashIcon,
  InfoCircledIcon,
  Link1Icon,
  CameraIcon,
  TagIcon,
  Spinner,
} from "./ui/icons";

// 统一的图片类型，可以接受管理界面和上传界面的两种不同图片对象
type ImageType = ImageFile | (ImageData & { status: 'success' });

interface ImageModalProps {
  image: ImageType | null;
  isOpen: boolean;
  onClose: () => void;
  onDelete?: (id: string) => Promise<void>;
  onTagsUpdate?: (id: string, tags: string[]) => void;
}

export default function ImageModal({ image, isOpen, onClose, onDelete, onTagsUpdate }: ImageModalProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [tags, setTags] = useState<string[]>([]);
  const [draftTags, setDraftTags] = useState("");
  const [isEditingTags, setIsEditingTags] = useState(false);
  const [isSavingTags, setIsSavingTags] = useState(false);
  const [tagError, setTagError] = useState("");

  useEffect(() => {
    if (!isOpen) {
      setShowDeleteConfirm(false);
      setIsDeleting(false);
      setIsEditingTags(false);
      setTagError("");
    }
  }, [isOpen]);

  useEffect(() => {
    const currentTags = image?.tags || [];
    setTags(currentTags);
    setDraftTags(currentTags.join(", "));
    setIsEditingTags(false);
    setTagError("");
  }, [image]);

  const handleDelete = async () => {
    if (!image || !onDelete || !image.id) return;

    try {
      setIsDeleting(true);
      await onDelete(image.id);
      setShowDeleteConfirm(false);  
      onClose();  
    } catch (err) {
      console.error("删除失败:", err);
    } finally {
      setIsDeleting(false);
    }
  };

  if (!image) return null;

  // 判断是否有可删除的功能
  const canDelete = onDelete && image.id;
  const imageId = getImageId(image);
  const canEditTags = Boolean(imageId);

  const handleStartEditTags = () => {
    setDraftTags(tags.join(", "));
    setTagError("");
    setIsEditingTags(true);
  };

  const handleCancelEditTags = () => {
    setDraftTags(tags.join(", "));
    setTagError("");
    setIsEditingTags(false);
  };

  const handleSaveTags = async () => {
    if (!imageId) return;

    const nextTags = normalizeTags(draftTags);
    try {
      setIsSavingTags(true);
      setTagError("");
      const response = await api.post<{ success: boolean; message: string; id: string; tags: string[] }>(
        "/api/update-tags",
        {
          id: imageId,
          tags: nextTags,
        }
      );

      setTags(response.tags);
      setDraftTags(response.tags.join(", "));
      setIsEditingTags(false);
      onTagsUpdate?.(response.id, response.tags);
    } catch (err) {
      console.error("更新标签失败:", err);
      setTagError("更新失败");
    } finally {
      setIsSavingTags(false);
    }
  };

  return (
    <AnimatePresence>
      {isOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
          onClick={onClose}
        >
          <div
            className="relative bg-white dark:bg-gray-900 rounded-2xl overflow-hidden w-full max-w-2xl max-h-[90vh] shadow-xl border border-gray-200 dark:border-gray-700"
            onClick={(e) => e.stopPropagation()}
          >
            {/* 标题栏 */}
            <div className="flex justify-between items-center px-6 py-5 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-3 flex-1 min-w-0">
                <CameraIcon className="h-6 w-6 text-blue-500" />
                <div className="flex-1 min-w-0">
                  <h3 className="text-lg font-semibold text-gray-900 dark:text-white truncate">{image.filename}</h3>
                  <p className="text-sm text-gray-500 dark:text-gray-400">图片详细信息</p>
                </div>
              </div>
              <button
                className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
                onClick={onClose}
              >
                <Cross1Icon className="h-5 w-5" />
              </button>
            </div>

            {/* 内容区域 */}
            <div className="overflow-y-auto max-h-[calc(90vh-12rem)] p-6 space-y-8">
              {/* 图片信息 */}
              <div>
                <div className="flex items-center gap-3 mb-5">
                  <InfoCircledIcon className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                  <h4 className="text-lg font-semibold text-gray-900 dark:text-white">图片信息</h4>
                </div>
                <ImageInfo image={image as any} />
              </div>

              {/* 图片标签 */}
              <div>
                <div className="flex items-center justify-between gap-3 mb-5">
                  <div className="flex items-center gap-3">
                    <TagIcon className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                    <h4 className="text-lg font-semibold text-gray-900 dark:text-white">标签</h4>
                  </div>
                  {canEditTags && !isEditingTags && (
                    <button
                      onClick={handleStartEditTags}
                      className="px-3 py-1.5 text-sm font-medium text-indigo-600 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-700 rounded-lg hover:bg-indigo-50 dark:hover:bg-indigo-900/20 transition-colors"
                    >
                      修改
                    </button>
                  )}
                </div>

                {isEditingTags ? (
                  <div className="space-y-3">
                    <input
                      value={draftTags}
                      onChange={(e) => setDraftTags(e.target.value)}
                      placeholder="nature, wallpaper"
                      className="w-full px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                      disabled={isSavingTags}
                    />
                    {tagError && (
                      <p className="text-sm text-red-600 dark:text-red-400">{tagError}</p>
                    )}
                    <div className="flex items-center gap-2">
                      <button
                        onClick={handleSaveTags}
                        disabled={isSavingTags}
                        className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 disabled:opacity-60 rounded-lg transition-colors"
                      >
                        {isSavingTags ? (
                          <Spinner className="h-4 w-4 text-white" />
                        ) : (
                          <CheckIcon className="h-4 w-4" />
                        )}
                        保存
                      </button>
                      <button
                        onClick={handleCancelEditTags}
                        disabled={isSavingTags}
                        className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 disabled:opacity-60 transition-colors"
                      >
                        <Cross1Icon className="h-4 w-4" />
                        取消
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="flex flex-wrap gap-2">
                    {tags.length > 0 ? (
                      tags.map((tag) => (
                        <span
                          key={tag}
                          className="px-2.5 py-1 text-sm text-indigo-700 dark:text-indigo-200 bg-indigo-50 dark:bg-indigo-900/30 border border-indigo-100 dark:border-indigo-800 rounded-full"
                        >
                          {tag}
                        </span>
                      ))
                    ) : (
                      <span className="text-sm text-gray-500 dark:text-gray-400">暂无标签</span>
                    )}
                  </div>
                )}
              </div>

              {/* 可用链接 */}
              <div>
                <div className="flex items-center gap-3 mb-5">
                  <Link1Icon className="h-5 w-5 text-green-600 dark:text-green-400" />
                  <h4 className="text-lg font-semibold text-gray-900 dark:text-white">可用链接</h4>
                </div>
                <ImageUrls image={image as any} />
              </div>
            </div>

            {/* 底部操作区域 */}
            <div className="flex justify-between items-center px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
              {canDelete && !showDeleteConfirm && (
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="px-4 py-2 text-sm font-medium text-red-600 dark:text-red-400 border border-red-200 dark:border-red-600 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                >
                  <TrashIcon className="h-4 w-4 mr-2 inline" />
                  删除图片
                </button>
              )}
              
              {showDeleteConfirm && (
                <div className="flex gap-2">
                  <DeleteConfirm
                    isDeleting={isDeleting}
                    onCancel={() => setShowDeleteConfirm(false)}
                    onConfirm={handleDelete}
                  />
                </div>
              )}
              
              {(!canDelete || !showDeleteConfirm) && <div />}
              
              <button
                onClick={onClose}
                className="px-6 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </AnimatePresence>
  );
}

function getImageId(image: ImageType): string {
  if (image.id) return image.id;
  if (!image.urls?.original) return "";

  const urlParts = image.urls.original.split("/");
  const filename = urlParts[urlParts.length - 1];
  return filename.includes(".") ? filename.split(".")[0] : "";
}

function normalizeTags(value: string): string[] {
  const seen = new Set<string>();
  const tags: string[] = [];

  value.split(",").forEach((rawTag) => {
    const tag = rawTag.trim();
    if (!tag || seen.has(tag)) return;
    seen.add(tag);
    tags.push(tag);
  });

  return tags;
}
