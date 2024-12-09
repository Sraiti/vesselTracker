import { useState, useEffect } from "react";
import {
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from "@/components/ui/command";
import { Location } from "../types/location";

interface LocationAutocompleteProps {
  value: Location | null;
  onChange: (location: Location | null) => void;
  placeholder?: string;
}

export default function LocationAutocomplete({
  value,
  onChange,
  placeholder,
}: Readonly<LocationAutocompleteProps>) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [locations, setLocations] = useState<Location[]>([]);

  const fetchLocations = async (query: string) => {
    if (query.length < 2) {
      setLocations([]);
      return;
    }

    try {
      const response = await fetch(
        `http://localhost:3058/autocomplete?text=${encodeURIComponent(query)}`
      );
      const data = await response.json();
      if (data.length > 0) {
        setLocations(data);
      } else {
        setLocations([]);
      }
    } catch (error) {
      console.error("Error fetching locations:", error);
      setLocations([]);
    }
  };

  // Close dropdown when clicking outside
  useEffect(() => {
    // const handleClick = (e: MouseEvent) => {
    //   console.log(e.target);
    //   if (e.target !== document.querySelector("#autocomplete-dropdown-inner")) {
    //     setOpen(false);
    //   }
    // };
    // document.addEventListener("click", handleClick);
    // return () => document.removeEventListener("click", handleClick);
  }, []);

  return (
    <Command className="relative" shouldFilter={false}>
      <div
        className="flex items-center border border-input rounded-md"
        // onClick={(e) => {
        //   e.stopPropagation();
        //   setOpen(true);
        // }}
      >
        {value?.country_code && (
          <span className="px-2 text-lg">
            {getFlagEmoji(value.country_code)}
          </span>
        )}
        <CommandInput
          value={value ? `${value.name} (${value.unlocode})` : search}
          onValueChange={(text) => {
            setSearch(text);
            onChange(null);
            fetchLocations(text);
          }}
          onFocus={() => setOpen(true)}
          placeholder={placeholder}
          className="flex h-10 w-full rounded-md bg-transparent px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none"
        />
      </div>

      <div
        id="autocomplete-dropdown-inner"
        className="rounded-md border bg-popover text-popover-foreground shadow-md overflow-hidden h-fit"
      >
        <CommandList>
          {locations.length === 0 && search.length >= 2 && (
            <CommandEmpty>No locations found.</CommandEmpty>
          )}
          {open && (
            <CommandGroup>
              {locations.map((location) => (
                <div
                  onSelect={() => {
                    onChange(location);
                    setOpen(false);
                    setSearch(location.name);
                  }}
                  onClick={(e) => {
                    e.stopPropagation();
                    e.preventDefault();
                    console.log("clicked", location.name);

                    onChange(location);
                    setOpen(false);
                    setSearch(location.name);
                  }}
                >
                  <CommandItem
                    key={location.id}
                    value={location.unlocode}
                    className="flex items-center gap-2 px-2 py-1.5 hover:bg-accent hover:text-accent-foreground cursor-pointer"
                  >
                    <span className="text-lg">
                      {getFlagEmoji(location.country_code)}
                    </span>
                    <span>{location.name}</span>
                    <span className="text-sm text-muted-foreground ml-0">
                      {location.is_port ? "⚓" : ""}
                    </span>
                    <span className="text-sm text-muted-foreground ml-auto">
                      {location.unlocode}
                    </span>
                  </CommandItem>
                </div>
              ))}
            </CommandGroup>
          )}
        </CommandList>
      </div>
    </Command>
  );
}

// Helper function to convert country code to flag emoji
function getFlagEmoji(countryCode: string) {
  const codePoints = countryCode
    .toUpperCase()
    .split("")
    .map((char) => 127397 + char.charCodeAt(0));
  return String.fromCodePoint(...codePoints);
}
