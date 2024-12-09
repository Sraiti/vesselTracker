import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { format } from "date-fns";
import { Anchor, Ship } from "lucide-react";
import { Schedule } from "../types/schedule";

interface ScheduleCardProps {
  schedule: Schedule;
  lastItem?: boolean;
  onSelect?: (schedule: Schedule) => void;
  isSelected?: boolean;
}

export function ScheduleCard({
  schedule,
  lastItem,
  onSelect,
  isSelected,
}: Readonly<ScheduleCardProps>) {
  return (
    <Card
      className={`w-screen max-w-lg hover:shadow-lg transition-shadow cursor-pointer ${
        lastItem ? "mb-10" : ""
      } ${isSelected ? "border-primary" : ""}`}
      onClick={() => onSelect?.(schedule)}
    >
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <div className="flex flex-col">
          <p className="text-sm font-medium leading-none">
            {schedule.OriginCity} ({schedule.OriginCountry})
          </p>
          <p className="text-sm text-muted-foreground">
            {format(new Date(schedule.DepartureDateTime), "PPP p")}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {schedule.LastKnownPosition.length > 0 ? (
            <div className="h-2 w-2 rounded-full bg-blue-500 animate-pulse" />
          ) : (
            <div className="h-2 w-2 rounded-full bg-muted-foreground" />
          )}
          <HoverCard>
            <HoverCardTrigger>
              <Ship className="h-4 w-4 text-muted-foreground" />
            </HoverCardTrigger>
            <span className="text-xs text-muted-foreground mr-1">
              {schedule.DepartureVesselName}
            </span>
            <HoverCardContent className="w-80">
              <div className="space-y-1">
                <h4 className="text-sm font-semibold">
                  {schedule.DepartureVesselName}
                </h4>
                <p className="text-sm">
                  IMO: {schedule.DepartureVesselIMONumber}
                  <br />
                  MMSI: {schedule.DepartureVesselMMSI}
                  <br />
                  Carrier Code: {schedule.DepartureVesselCarrierCode}
                </p>
              </div>
            </HoverCardContent>
          </HoverCard>
        </div>
      </CardHeader>
      <CardContent>
        <div className="flex items-center space-x-2">
          <div className="flex-1 space-y-1">
            <div className="flex items-center relative">
              <div className="text-primary bg-background/95 p-1 rounded-full shadow-lg">
                <Anchor className="h-3 w-3" />
              </div>
              <div className="h-px flex-1 bg-muted mx-2 relative">
                {schedule.TransportLegs.length > 1 && (
                  <HoverCard>
                    <HoverCardTrigger>
                      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-muted-foreground text-background w-5 h-5 rounded-full flex items-center justify-center text-xs cursor-pointer">
                        {schedule.TransportLegs.length - 1}
                      </div>
                    </HoverCardTrigger>
                    <HoverCardContent className="w-80">
                      <div className="space-y-2">
                        {schedule.TransportLegs.map((leg, index) => (
                          <div key={index} className="text-sm space-y-1">
                            <div className="font-medium flex items-center gap-2">
                              <Ship className="h-4 w-4" />
                              {leg.VesselName || "Land Transport"}
                            </div>
                            <div className="grid grid-cols-2 gap-1 text-xs">
                              <div>
                                <p className="text-muted-foreground">
                                  Departs:
                                </p>
                                <p>
                                  {format(
                                    new Date(leg.DepartureDateTime),
                                    "PPP p"
                                  )}
                                </p>
                                <p className="text-muted-foreground mt-1">
                                  {leg.OriginName}
                                </p>
                              </div>
                              <div>
                                <p className="text-muted-foreground">
                                  Arrives:
                                </p>
                                <p>
                                  {format(
                                    new Date(leg.ArrivalDateTime),
                                    "PPP p"
                                  )}
                                </p>
                                <p className="text-muted-foreground mt-1">
                                  {leg.DestinationName}
                                </p>
                              </div>
                            </div>
                            {index < schedule.TransportLegs.length - 1 && (
                              <div className="h-px bg-border my-2" />
                            )}
                          </div>
                        ))}
                      </div>
                    </HoverCardContent>
                  </HoverCard>
                )}
              </div>
              <div className="text-destructive bg-background/95 p-2 rounded-full shadow-lg">
                <Anchor className="h-3 w-3" />
              </div>
            </div>
            <div className="flex justify-between text-sm">
              <span>{schedule.OriginName}</span>
              <span>{schedule.DestinationName}</span>
            </div>
            <p className="text-xs text-muted-foreground text-right">
              Arrives: {format(new Date(schedule.ArrivalDateTime), "PPP p")}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
