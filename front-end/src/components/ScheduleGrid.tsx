import { Schedule, ScheduleResponse } from "@/types/schedule";
import { ScheduleCard } from "./ScheduleCard";

interface ScheduleGridProps {
  data: ScheduleResponse;
  onSelect?: (schedule: Schedule) => void;
}

export function ScheduleGrid({ data, onSelect }: ScheduleGridProps) {
  return (
    <div className="grid grid-rows-1 md:grid-rows-2 lg:grid-rows-3 gap-3 p-4 justify-center w-full">
      {data.schedules.map((schedule, index) => (
        <ScheduleCard
          key={`${schedule.CarrierProductID}-${index}`}
          lastItem={index === data.schedules.length - 1}
          schedule={schedule}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}
