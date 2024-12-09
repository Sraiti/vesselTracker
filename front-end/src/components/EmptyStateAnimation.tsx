import { Ship, Waves } from "lucide-react";

interface EmptyStateAnimationProps {
  isLoading?: boolean;
  message?: string;
}

export default function EmptyStateAnimation({
  isLoading,
  message,
}: EmptyStateAnimationProps) {
  return (
    <div className="flex flex-col items-center justify-center p-8 text-muted-foreground rounded-lg">
      <div className="relative w-32 h-32 flex items-center justify-center">
        {/* Waves animation - positioned below ship */}

        {/* Ship animation - positioned above waves */}
        <div
          className={`absolute transform ${"animate-[float_3s_ease-in-out_infinite]"}`}
        >
          <Ship className="w-32 h-32 text-white" />
        </div>
      </div>

      <p className="mt-4 text-sm text-white">
        {message || (isLoading ? "Searching routes..." : "No routes found")}
      </p>
    </div>
  );
}
