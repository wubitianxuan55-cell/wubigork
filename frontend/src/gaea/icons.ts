/* eslint-disable @typescript-eslint/no-explicit-any -- antd 图标类型(AntdIconProps)与 SVGProps 不兼容，
   wrap 桥接层需 any 类型放宽；图标组件运行时行为不受影响 */
// gaea 图标兼容层 — lucide-react → @ant-design/icons 统一映射。
// 目的：办公模块与主应用共享同一套图标体系（antd），消除两套图标来源。
// 用法：`import { X, Check, ... } from "../icons"`（原 lucide 图标名保持可用）。
import {
  AlertOutlined,
  ApartmentOutlined,
  ApiOutlined,
  AppstoreOutlined,
  ArrowDownOutlined,
  ArrowUpOutlined,
  AuditOutlined,
  BarChartOutlined,
  BarsOutlined,
  BlockOutlined,
  BookOutlined,
  BulbOutlined,
  CheckCircleOutlined,
  CheckOutlined,
  ClockCircleOutlined,
  CloseOutlined,
  CloudUploadOutlined,
  CodeOutlined,
  CopyOutlined,
  DeleteOutlined,
  DownOutlined,
  EditOutlined,
  EyeOutlined,
  FileExcelOutlined,
  FileImageOutlined,
  FileOutlined,
  FileTextOutlined,
  FolderAddOutlined,
  FolderOpenOutlined,
  FolderOutlined,
  GlobalOutlined,
  HolderOutlined,
  HomeOutlined,
  LineChartOutlined,
  LinkOutlined,
  LoadingOutlined,
  MenuOutlined,
  MessageOutlined,
  MobileOutlined,
  MoneyCollectOutlined,
  MoonOutlined,
  NumberOutlined,
  PaperClipOutlined,
  PartitionOutlined,
  PictureOutlined,
  PlusOutlined,
  PullRequestOutlined,
  QrcodeOutlined,
  ReloadOutlined,
  RightOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  SaveOutlined,
  SearchOutlined,
  SettingOutlined,
  StopOutlined,
  SunOutlined,
  SwapOutlined,
  TableOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  ToolOutlined,
  WalletOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { createElement, type ComponentType, type SVGProps } from "react";

export type IconProps = SVGProps<SVGSVGElement> & { size?: number | string };
export type Icon = ComponentType<IconProps>;

// 包装 antd 图标，兼容 lucide 的 size prop（映射到 style.fontSize）。
function wrap(Comp: ComponentType<any>): Icon {
  return function GaeaIcon(props: IconProps) {
    const { size, style, ...rest } = props;
    const fontSize = size !== undefined ? size : undefined;
    return createElement(Comp as ComponentType<any>, { ...(rest as object), style: fontSize !== undefined ? { ...style, fontSize } : style });
  };
}

// lucide 名 → antd 图标
export const AlertCircle: Icon = wrap(AlertOutlined);
export const ArrowDown: Icon = wrap(ArrowDownOutlined);
export const ArrowUp: Icon = wrap(ArrowUpOutlined);
export const BarChart3: Icon = wrap(BarChartOutlined);
export const BookOpen: Icon = wrap(BookOutlined);
export const Bot: Icon = wrap(RobotOutlined);
export const Box: Icon = wrap(BlockOutlined);
export const Brain: Icon = wrap(ApiOutlined);
export const Check: Icon = wrap(CheckOutlined);

export const CheckCircle: Icon = wrap(CheckCircleOutlined);
export const ChevronDown: Icon = wrap(DownOutlined);
export const ChevronRight: Icon = wrap(RightOutlined);
export const ChevronsUpDown: Icon = wrap(SwapOutlined);
export const Circle: Icon = wrap(CheckCircleOutlined);
export const ClipboardList: Icon = wrap(AuditOutlined);
export const Clock: Icon = wrap(ClockCircleOutlined);
export const CloudUpload: Icon = wrap(CloudUploadOutlined);
export const Coins: Icon = wrap(MoneyCollectOutlined);
export const Command: Icon = wrap(CodeOutlined);
export const Copy: Icon = wrap(CopyOutlined);
export const Cpu: Icon = wrap(ApiOutlined);
export const ExternalLink: Icon = wrap(LinkOutlined);
export const Eye: Icon = wrap(EyeOutlined);
export const File: Icon = wrap(FileOutlined);
export const FileImage: Icon = wrap(FileImageOutlined);
export const FileSpreadsheet: Icon = wrap(FileExcelOutlined);
export const FileText: Icon = wrap(FileTextOutlined);
export const Folder: Icon = wrap(FolderOutlined);
export const FolderGit2: Icon = wrap(FolderOpenOutlined);
export const FolderOpen: Icon = wrap(FolderOpenOutlined);
export const FolderPlus: Icon = wrap(FolderAddOutlined);
export const GitBranch: Icon = wrap(PullRequestOutlined);
export const Globe: Icon = wrap(GlobalOutlined);
export const Home: Icon = wrap(HomeOutlined);
export const Image: Icon = wrap(PictureOutlined);
export const Loader: Icon = wrap(LoadingOutlined);
export const MessageSquare: Icon = wrap(MessageOutlined);
export const Moon: Icon = wrap(MoonOutlined);
export const Palette: Icon = wrap(EditOutlined);
export const Paperclip: Icon = wrap(PaperClipOutlined);
export const Pencil: Icon = wrap(EditOutlined);
export const Plug: Icon = wrap(ThunderboltOutlined);
export const Plus: Icon = wrap(PlusOutlined);
export const Puzzle: Icon = wrap(ApartmentOutlined);
export const QrCode: Icon = wrap(QrcodeOutlined);
export const RefreshCw: Icon = wrap(ReloadOutlined);
export const Save: Icon = wrap(SaveOutlined);
export const ScrollText: Icon = wrap(FileTextOutlined);
export const Search: Icon = wrap(SearchOutlined);
export const Shield: Icon = wrap(SafetyCertificateOutlined);
export const ShieldAlert: Icon = wrap(WarningOutlined);
export const Smartphone: Icon = wrap(MobileOutlined);
export const Square: Icon = wrap(AppstoreOutlined);
export const Sun: Icon = wrap(SunOutlined);
export const Trash2: Icon = wrap(DeleteOutlined);
export const TrendingUp: Icon = wrap(LineChartOutlined);
export const Wallet: Icon = wrap(WalletOutlined);
export const Wrench: Icon = wrap(ToolOutlined);
export const X: Icon = wrap(CloseOutlined);
export const XasXIcon: Icon = wrap(CloseOutlined);

export const SquarePen: Icon = wrap(EditOutlined);
export const FolderTree: Icon = wrap(FolderOpenOutlined);
export const PanelRightOpen: Icon = wrap(HolderOutlined);
export const PanelRightClose: Icon = wrap(HolderOutlined);
export const Settings: Icon = wrap(SettingOutlined);

export const Blocks: Icon = wrap(ApartmentOutlined);
export const PanelLeftClose: Icon = wrap(MenuOutlined);
export const PanelLeftOpen: Icon = wrap(MenuOutlined);
export const Ban: Icon = wrap(StopOutlined);
export const Loader2: Icon = wrap(LoadingOutlined);

export const Calculator: Icon = wrap(NumberOutlined);
export const FilePen: Icon = wrap(EditOutlined);
export const Hourglass: Icon = wrap(ClockCircleOutlined);
export const Layers: Icon = wrap(AppstoreOutlined);
export const ListTree: Icon = wrap(PartitionOutlined);
export const PlusCircle: Icon = wrap(PlusOutlined);
export const Sparkles: Icon = wrap(BulbOutlined);
export const Users: Icon = wrap(TeamOutlined);
export const List: Icon = wrap(BarsOutlined);
export const Table: Icon = wrap(TableOutlined);
export const Zap: Icon = wrap(ThunderboltOutlined);
